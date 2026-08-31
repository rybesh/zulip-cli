package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rybesh/zulip-cli/types"
)

const (
	APIVersion = "v1"

	// DefaultTimeout bounds a single request. Long-polling for events needs
	// longer and sets its own bound; see GetEvents.
	DefaultTimeout = 15 * time.Second
)

// Client is the main Zulip API client
type Client struct {
	BaseURL    string
	Email      string
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
	Verbose    bool
	// Timeout bounds a single request. Zero means DefaultTimeout; a negative
	// value means no timeout at all.
	Timeout time.Duration
	// Warnf receives warnings about requests that otherwise succeeded, such as
	// parameters the server ignored. Nil writes them to stderr; set it to a
	// no-op to silence them.
	Warnf func(format string, args ...interface{})
}

// warnf reports a problem that did not fail the request.
func (c *Client) warnf(format string, args ...interface{}) {
	if c.Warnf != nil {
		c.Warnf(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// Config holds configuration for creating a new client
type Config struct {
	URL      string
	Email    string
	APIKey   string
	Insecure bool
	// CertBundle is a PEM file of certificate authorities to trust instead of
	// the system roots, for servers using a private CA.
	CertBundle string
	// ClientCert and ClientCertKey are a PEM certificate/key pair to present
	// when the server asks for a client certificate. Both or neither.
	ClientCert    string
	ClientCertKey string
	// Timeout bounds a single request. Zero means DefaultTimeout.
	Timeout time.Duration
	Verbose bool
}

// APIError is an error response from the Zulip server, as opposed to a failure
// to reach it. Callers can inspect Code to react to a specific condition; see
// APIErrorCode.
type APIError struct {
	StatusCode int
	Code       string
	Msg        string
	Body       string
}

func (e *APIError) Error() string {
	switch {
	case e.Msg == "":
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
	case e.StatusCode == http.StatusUnauthorized:
		// Credentials are supplied by the environment, so name them: this is
		// the first point at which a wrong key becomes visible.
		return fmt.Sprintf("API error: %s (check ZULIP_EMAIL and ZULIP_API_KEY)", e.Msg)
	default:
		return fmt.Sprintf("API error: %s", e.Msg)
	}
}

// APIErrorCode returns the Zulip error code carried by err, or "" if err did
// not come from a server error response.
func APIErrorCode(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// FormFile is a file to send as part of a multipart request.
type FormFile struct {
	// Filename is the name the server records. It should be a bare file name,
	// not the local path the user happened to type.
	Filename string
	Reader   io.Reader
}

// NewClient creates a new Zulip API client from environment variables
func NewClient() (*Client, error) {
	cfg := Config{
		URL:           os.Getenv("ZULIP_URL"),
		Email:         os.Getenv("ZULIP_EMAIL"),
		APIKey:        os.Getenv("ZULIP_API_KEY"),
		CertBundle:    os.Getenv("ZULIP_CERT_BUNDLE"),
		ClientCert:    os.Getenv("ZULIP_CLIENT_CERT"),
		ClientCertKey: os.Getenv("ZULIP_CLIENT_CERT_KEY"),
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("ZULIP_URL environment variable is required")
	}
	if cfg.Email == "" {
		return nil, fmt.Errorf("ZULIP_EMAIL environment variable is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ZULIP_API_KEY environment variable is required")
	}

	if v := os.Getenv("ZULIP_INSECURE"); v != "" {
		insecure, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ZULIP_INSECURE %q: want a boolean", v)
		}
		cfg.Insecure = insecure
	}

	if v := os.Getenv("ZULIP_TIMEOUT"); v != "" {
		timeout, err := ParseTimeout(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ZULIP_TIMEOUT: %w", err)
		}
		cfg.Timeout = timeout
	}

	return NewClientWithConfig(cfg)
}

// ParseTimeout reads a request timeout written either as a Go duration ("45s",
// "2m") or as a bare number of seconds ("45").
func ParseTimeout(s string) (time.Duration, error) {
	if timeout, err := time.ParseDuration(s); err == nil {
		return timeout, nil
	}
	seconds, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a duration such as 45s or a number of seconds", s)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

// NewClientWithConfig creates a new Zulip API client with custom configuration
func NewClientWithConfig(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	if cfg.Email == "" {
		return nil, fmt.Errorf("Email is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Normalize URL
	baseURL := cfg.URL
	if strings.HasPrefix(baseURL, "localhost") {
		baseURL = "http://" + baseURL
	} else if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api") {
		baseURL += "/api"
	}
	baseURL += "/"

	tlsConfig, err := tlsConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Timeouts are applied per request, so that long-polling can raise its own
	// bound without disturbing the shared HTTP client.
	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	userAgent := fmt.Sprintf("ZulipGo/%s (%s; %s)", ClientVersion, runtime.GOOS, runtime.GOARCH)

	return &Client{
		BaseURL:    baseURL,
		Email:      cfg.Email,
		APIKey:     cfg.APIKey,
		HTTPClient: httpClient,
		UserAgent:  userAgent,
		Verbose:    cfg.Verbose,
		Timeout:    timeout,
	}, nil
}

// tlsConfig builds the TLS settings for the configured trust and client
// certificate options.
func tlsConfig(cfg Config) (*tls.Config, error) {
	conf := &tls.Config{InsecureSkipVerify: cfg.Insecure}

	if cfg.CertBundle != "" {
		pemBytes, err := os.ReadFile(cfg.CertBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate bundle: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("no certificates found in certificate bundle %s", cfg.CertBundle)
		}
		conf.RootCAs = pool
	}

	switch {
	case cfg.ClientCert != "" && cfg.ClientCertKey == "":
		return nil, fmt.Errorf("client certificate given without its key")
	case cfg.ClientCert == "" && cfg.ClientCertKey != "":
		return nil, fmt.Errorf("client certificate key given without a certificate")
	case cfg.ClientCert != "":
		cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientCertKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		conf.Certificates = []tls.Certificate{cert}
	}

	return conf, nil
}

// requestTimeout is the bound for an ordinary request.
func (c *Client) requestTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultTimeout
	}
	return c.Timeout
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, endpoint string, params map[string]interface{}, files map[string]FormFile) ([]byte, error) {
	return c.doRequestContext(context.Background(), c.requestTimeout(), method, endpoint, params, files)
}

// doRequestContext performs an authenticated request, giving up after timeout
// unless ctx is cancelled first. A timeout of zero or less means no bound.
func (c *Client) doRequestContext(ctx context.Context, timeout time.Duration, method, endpoint string, params map[string]interface{}, files map[string]FormFile) ([]byte, error) {
	fullURL := c.BaseURL + APIVersion + "/" + endpoint

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var req *http.Request
	var err error

	if len(files) > 0 {
		// Multipart form data for file uploads
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add parameters
		for key, val := range params {
			var strVal string
			switch v := val.(type) {
			case string:
				strVal = v
			default:
				jsonBytes, _ := json.Marshal(v)
				strVal = string(jsonBytes)
			}
			writer.WriteField(key, strVal)
		}

		// Add files
		for fieldName, file := range files {
			part, err := writer.CreateFormFile(fieldName, file.Filename)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(part, file.Reader); err != nil {
				return nil, err
			}
		}

		writer.Close()
		req, err = http.NewRequestWithContext(ctx, method, fullURL, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
	} else if method == "GET" || method == "DELETE" {
		// Query parameters
		if len(params) > 0 {
			values := url.Values{}
			for key, val := range params {
				switch v := val.(type) {
				case string:
					values.Add(key, v)
				default:
					jsonBytes, _ := json.Marshal(v)
					values.Add(key, string(jsonBytes))
				}
			}
			fullURL += "?" + values.Encode()
		}
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
	} else {
		// Form data for POST/PATCH/PUT
		values := url.Values{}
		for key, val := range params {
			switch v := val.(type) {
			case string:
				values.Add(key, v)
			default:
				jsonBytes, _ := json.Marshal(v)
				values.Add(key, string(jsonBytes))
			}
		}
		req, err = http.NewRequestWithContext(ctx, method, fullURL, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return nil, err
	}

	// Set authentication
	req.SetBasicAuth(c.Email, c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "Request: %s %s\n", method, fullURL)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "Response: %d %s\n", resp.StatusCode, string(body))
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(body)}
		var errResp types.Response
		if err := json.Unmarshal(body, &errResp); err == nil {
			apiErr.Msg = errResp.Msg
			apiErr.Code = errResp.Code
		}
		return body, apiErr
	}

	// Zulip accepts parameters it does not recognize and names them in the
	// success response. Surfacing that keeps a renamed or removed parameter
	// from failing invisibly. The cheap substring check keeps large responses
	// from being parsed twice for nothing.
	if bytes.Contains(body, []byte("ignored_parameters_unsupported")) {
		var meta types.Response
		if err := json.Unmarshal(body, &meta); err == nil && len(meta.IgnoredParameters) > 0 {
			c.warnf("server ignored unsupported parameters on %s: %s",
				endpoint, strings.Join(meta.IgnoredParameters, ", "))
		}
	}

	return body, nil
}

// Get performs a GET request
func (c *Client) Get(endpoint string, params map[string]interface{}) ([]byte, error) {
	return c.doRequest("GET", endpoint, params, nil)
}

// Post performs a POST request
func (c *Client) Post(endpoint string, params map[string]interface{}) ([]byte, error) {
	return c.doRequest("POST", endpoint, params, nil)
}

// Patch performs a PATCH request
func (c *Client) Patch(endpoint string, params map[string]interface{}) ([]byte, error) {
	return c.doRequest("PATCH", endpoint, params, nil)
}

// Delete performs a DELETE request
func (c *Client) Delete(endpoint string, params map[string]interface{}) ([]byte, error) {
	return c.doRequest("DELETE", endpoint, params, nil)
}

// PostWithFiles performs a POST request with file uploads
func (c *Client) PostWithFiles(endpoint string, params map[string]interface{}, files map[string]FormFile) ([]byte, error) {
	return c.doRequest("POST", endpoint, params, files)
}

// GetServerSettings fetches server settings
func (c *Client) GetServerSettings() (*types.ServerSettings, error) {
	body, err := c.Get("server_settings", nil)
	if err != nil {
		return nil, err
	}

	var settings types.ServerSettings
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}
