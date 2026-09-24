package client

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
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

// testRoutedServer answers each endpoint with the body mapped to its path, and
// a bare success for anything else. A flow that has to ask the server something
// before it can act needs more than one canned answer.
func testRoutedServer(t *testing.T, bodies map[string]string) (*Client, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		body, found := bodies[r.URL.Path]
		if !found {
			body = `{"result":"success"}`
		}
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

// upload is what uploadServer saw of the first uploaded part.
type upload struct {
	Field, Filename, ContentType string
	Content                      []byte
}

// uploadServer records the first uploaded part.
func uploadServer(t *testing.T, body string) (*Client, func() upload) {
	t.Helper()
	var mu sync.Mutex
	var got upload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		} else {
			mu.Lock()
			for name, headers := range r.MultipartForm.File {
				got = upload{Field: name, Filename: headers[0].Filename, ContentType: headers[0].Header.Get("Content-Type")}
				if f, err := headers[0].Open(); err == nil {
					got.Content, _ = io.ReadAll(f)
					f.Close()
				}
			}
			mu.Unlock()
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	return testClient(t, Config{URL: srv.URL}), func() upload {
		mu.Lock()
		defer mu.Unlock()
		return got
	}
}

// The path the user typed is not the name the server should store.
func TestUploadFileSendsBaseName(t *testing.T) {
	c, uploaded := uploadServer(t, `{"result":"success","uri":"/user_uploads/1/x/q3.pdf"}`)

	if _, err := c.UploadFile(strings.NewReader("data"), "/home/me/docs/q3.pdf"); err != nil {
		t.Fatal(err)
	}

	got := uploaded()
	field, filename := got.Field, got.Filename
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

	got := uploaded()
	field, filename := got.Field, got.Filename
	if field != "file" {
		t.Errorf("field name = %q, want file", field)
	}
	// The extension is how the server recognizes the image format.
	if filename != "smiley.png" {
		t.Errorf("stored file name = %q, want smiley.png", filename)
	}
}

// pngHeader is enough of a PNG for content sniffing to recognize it.
var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

// The server stores the part's type with the upload and serves the file back
// with it, so an image sent as application/octet-stream gets no preview.
func TestUploadFileSendsContentType(t *testing.T) {
	tests := []struct {
		name, filename string
		content        []byte
		want           string
	}{
		{"from extension", "photo.jpg", []byte("not really a jpeg"), "image/jpeg"},
		{"extension case", "PHOTO.PNG", []byte("x"), "image/png"},
		{"sniffed without extension", "photo", pngHeader, "image/png"},
		{"sniffed unknown extension", "photo.zzunknown", pngHeader, "image/png"},
		{"unrecognizable", "blob", []byte{0x00, 0x01, 0x02, 0xff}, "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, uploaded := uploadServer(t, `{"result":"success","uri":"/user_uploads/1/x/f"}`)

			if _, err := c.UploadFile(bytes.NewReader(tt.content), tt.filename); err != nil {
				t.Fatal(err)
			}

			got := uploaded()
			if got.ContentType != tt.want {
				t.Errorf("Content-Type = %q, want %q", got.ContentType, tt.want)
			}
			// Sniffing reads ahead; the whole file must still arrive.
			if !bytes.Equal(got.Content, tt.content) {
				t.Errorf("content = %q, want %q", got.Content, tt.content)
			}
		})
	}
}

func TestUploadCustomEmojiSendsContentType(t *testing.T) {
	c, uploaded := uploadServer(t, `{"result":"success"}`)

	if _, err := c.UploadCustomEmoji("smiley", "../art/smiley.png", bytes.NewReader(pngHeader)); err != nil {
		t.Fatal(err)
	}

	if got := uploaded().ContentType; got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
}

// The part header is built with CreatePart rather than CreateFormFile, so a
// name that needs escaping must still reach the server intact.
func TestUploadFileEscapesFilename(t *testing.T) {
	c, uploaded := uploadServer(t, `{"result":"success","uri":"/user_uploads/1/x/f"}`)

	name := `say "hi" \ bye.txt`
	if _, err := c.UploadFile(strings.NewReader("hi"), name); err != nil {
		t.Fatal(err)
	}

	if got := uploaded().Filename; got != name {
		t.Errorf("stored file name = %q, want %q", got, name)
	}
}
