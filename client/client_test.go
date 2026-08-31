package client

import (
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// recorder captures what a fake Zulip server received.
type recorder struct {
	mu     sync.Mutex
	method string
	path   string
	params url.Values
	calls  int
}

func (r *recorder) record(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Form holds the query string and, for form-encoded bodies, the body too.
	_ = req.ParseForm()
	r.method, r.path, r.params = req.Method, req.URL.Path, req.Form
	r.calls++
}

func (r *recorder) snapshot() (method, path string, params url.Values, calls int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.method, r.path, r.params, r.calls
}

// testClient builds a client for cfg, filling in credentials and silencing
// warnings so they do not land on the test's stderr.
func testClient(t *testing.T, cfg Config) *Client {
	t.Helper()
	if cfg.Email == "" {
		cfg.Email = "bot@example.com"
	}
	if cfg.APIKey == "" {
		cfg.APIKey = "key"
	}
	c, err := NewClientWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewClientWithConfig: %v", err)
	}
	c.Warnf = func(string, ...interface{}) {}
	return c
}

// testServer answers every request with body and records what it received.
func testServer(t *testing.T, body string) (*Client, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return testClient(t, Config{URL: srv.URL}), rec
}

func TestBaseURLNormalization(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"localhost:9991", "http://localhost:9991/api/"},
		{"example.zulipchat.com", "https://example.zulipchat.com/api/"},
		{"https://example.zulipchat.com", "https://example.zulipchat.com/api/"},
		{"https://example.zulipchat.com/", "https://example.zulipchat.com/api/"},
		{"https://example.zulipchat.com/api", "https://example.zulipchat.com/api/"},
		{"http://127.0.0.1:8099", "http://127.0.0.1:8099/api/"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			c := testClient(t, Config{URL: tc.in})
			if c.BaseURL != tc.want {
				t.Errorf("BaseURL = %q, want %q", c.BaseURL, tc.want)
			}
		})
	}
}

func TestNewClientWithConfigRequiresCredentials(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"no URL", Config{Email: "bot@example.com", APIKey: "key"}},
		{"no email", Config{URL: "example.com", APIKey: "key"}},
		{"no API key", Config{URL: "example.com", Email: "bot@example.com"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewClientWithConfig(tc.cfg); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// Constructing a client used to probe the server, costing a round trip on every
// invocation while proving nothing about the credentials.
func TestNewClientDoesNotContactServer(t *testing.T) {
	c, rec := testServer(t, `{"result":"success"}`)
	_ = c

	if _, _, _, calls := rec.snapshot(); calls != 0 {
		t.Fatalf("constructor made %d requests, want 0", calls)
	}
}

func TestParseTimeout(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{in: "45s", want: 45 * time.Second},
		{in: "2m", want: 2 * time.Minute},
		{in: "45", want: 45 * time.Second},
		{in: "0.5", want: 500 * time.Millisecond},
		{in: "soon", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseTimeout(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseTimeout(%q) = %v, want an error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTimeout(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseTimeout(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestTimeoutIsConfigurable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprint(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	slow := testClient(t, Config{URL: srv.URL, Timeout: 20 * time.Millisecond})
	if _, err := slow.GetProfile(); err == nil {
		t.Fatal("expected the short timeout to cut the request off")
	}

	patient := testClient(t, Config{URL: srv.URL, Timeout: 5 * time.Second})
	if _, err := patient.GetProfile(); err != nil {
		t.Fatalf("request with a generous timeout failed: %v", err)
	}
}

func TestCertBundleTrustsPrivateCA(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	untrusted := testClient(t, Config{URL: srv.URL})
	if _, err := untrusted.GetProfile(); err == nil {
		t.Fatal("expected the server's own certificate to be untrusted")
	}

	bundle := filepath.Join(t.TempDir(), "ca.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(bundle, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	trusted := testClient(t, Config{URL: srv.URL, CertBundle: bundle})
	if _, err := trusted.GetProfile(); err != nil {
		t.Fatalf("request with the CA bundle failed: %v", err)
	}

	insecure := testClient(t, Config{URL: srv.URL, Insecure: true})
	if _, err := insecure.GetProfile(); err != nil {
		t.Fatalf("request with verification disabled failed: %v", err)
	}
}

func TestTLSConfigErrors(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"certificate without key", Config{URL: "example.com", ClientCert: "cert.pem"}},
		{"key without certificate", Config{URL: "example.com", ClientCertKey: "key.pem"}},
		{"missing bundle file", Config{URL: "example.com", CertBundle: "/no/such/ca.pem"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.Email, cfg.APIKey = "bot@example.com", "key"
			if _, err := NewClientWithConfig(cfg); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestAPIErrorCarriesCodeAndCredentialHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"result":"error","msg":"Invalid API key","code":"UNAUTHORIZED"}`)
	}))
	defer srv.Close()

	c := testClient(t, Config{URL: srv.URL})
	_, err := c.GetProfile()

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %T (%v), want *APIError", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if APIErrorCode(err) != "UNAUTHORIZED" {
		t.Errorf("APIErrorCode = %q, want UNAUTHORIZED", APIErrorCode(err))
	}
	// A wrong key is an environment problem, so the message says which.
	if !strings.Contains(err.Error(), "ZULIP_API_KEY") {
		t.Errorf("error %q does not name the credentials to check", err)
	}
}

func TestAPIErrorWithoutJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "<html>gateway blew up</html>")
	}))
	defer srv.Close()

	c := testClient(t, Config{URL: srv.URL})
	_, err := c.GetProfile()
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("got %v, want an HTTP 502 error", err)
	}
	if APIErrorCode(err) != "" {
		t.Errorf("APIErrorCode = %q, want empty", APIErrorCode(err))
	}
}

func TestIgnoredParametersAreSurfaced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"result":"success","streams":[],`+
			`"ignored_parameters_unsupported":["bogus","stream_post_policy"]}`)
	}))
	defer srv.Close()

	c := testClient(t, Config{URL: srv.URL})
	var warnings []string
	c.Warnf = func(format string, args ...interface{}) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	resp, err := c.GetStreams(GetStreamsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "bogus, stream_post_policy") {
		t.Fatalf("warnings = %v, want one naming both parameters", warnings)
	}
	if got := resp.IgnoredParameters; len(got) != 2 {
		t.Errorf("IgnoredParameters = %v, want both parameters on the response", got)
	}
}

func TestNoWarningWhenEveryParameterIsProcessed(t *testing.T) {
	c, _ := testServer(t, `{"result":"success","streams":[]}`)
	warned := false
	c.Warnf = func(string, ...interface{}) { warned = true }

	if _, err := c.GetStreams(GetStreamsRequest{}); err != nil {
		t.Fatal(err)
	}
	if warned {
		t.Fatal("warned about nothing")
	}
}

// uploadServer records the field name and file name of the first uploaded part.
func uploadServer(t *testing.T, body string) (*Client, func() (field, filename string)) {
	t.Helper()
	var mu sync.Mutex
	var field, filename string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		} else {
			mu.Lock()
			for name, headers := range r.MultipartForm.File {
				field, filename = name, headers[0].Filename
			}
			mu.Unlock()
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	return testClient(t, Config{URL: srv.URL}), func() (string, string) {
		mu.Lock()
		defer mu.Unlock()
		return field, filename
	}
}

// The path the user typed is not the name the server should store.
func TestUploadFileSendsBaseName(t *testing.T) {
	c, uploaded := uploadServer(t, `{"result":"success","uri":"/user_uploads/1/x/q3.pdf"}`)

	if _, err := c.UploadFile(strings.NewReader("data"), "/home/me/docs/q3.pdf"); err != nil {
		t.Fatal(err)
	}

	field, filename := uploaded()
	if field != "file" {
		t.Errorf("field name = %q, want file", field)
	}
	if filename != "q3.pdf" {
		t.Errorf("stored file name = %q, want q3.pdf", filename)
	}
}

func TestUploadCustomEmojiSendsBaseName(t *testing.T) {
	c, uploaded := uploadServer(t, `{"result":"success"}`)

	if _, err := c.UploadCustomEmoji("smiley", "../art/smiley.png", strings.NewReader("png")); err != nil {
		t.Fatal(err)
	}

	field, filename := uploaded()
	if field != "file" {
		t.Errorf("field name = %q, want file", field)
	}
	// The extension is how the server recognizes the image format.
	if filename != "smiley.png" {
		t.Errorf("stored file name = %q, want smiley.png", filename)
	}
}
