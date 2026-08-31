package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/rybesh/zulip-cli/types"
)

const (
	APIVersion   = "v1"
	ClientVersion = "0.1.0"
)

// Client is the main Zulip API client
type Client struct {
	BaseURL    string
	Email      string
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
	Verbose    bool
}

// Config holds configuration for creating a new client
type Config struct {
	URL           string
	Email         string
	APIKey        string
	Insecure      bool
	CertBundle    string
	ClientCert    string
	ClientCertKey string
	Verbose       bool
}

// NewClient creates a new Zulip API client from environment variables
func NewClient() (*Client, error) {
	url := os.Getenv("ZULIP_URL")
	email := os.Getenv("ZULIP_EMAIL")
	apiKey := os.Getenv("ZULIP_API_KEY")

	if url == "" {
		return nil, fmt.Errorf("ZULIP_URL environment variable is required")
	}
	if email == "" {
		return nil, fmt.Errorf("ZULIP_EMAIL environment variable is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ZULIP_API_KEY environment variable is required")
	}

	return NewClientWithConfig(Config{
		URL:    url,
		Email:  email,
		APIKey: apiKey,
	})
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

	// Configure HTTP client
	transport := &http.Transport{}
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	userAgent := fmt.Sprintf("ZulipGo/%s (%s; %s)", ClientVersion, runtime.GOOS, runtime.GOARCH)

	client := &Client{
		BaseURL:    baseURL,
		Email:      cfg.Email,
		APIKey:     cfg.APIKey,
		HTTPClient: httpClient,
		UserAgent:  userAgent,
		Verbose:    cfg.Verbose,
	}

	// Verify connection by fetching server settings
	_, err := client.GetServerSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return client, nil
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, endpoint string, params map[string]interface{}, files map[string]io.Reader) ([]byte, error) {
	fullURL := c.BaseURL + APIVersion + "/" + endpoint

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
			part, err := writer.CreateFormFile(fieldName, fieldName)
			if err != nil {
				return nil, err
			}
			io.Copy(part, file)
		}

		writer.Close()
		req, err = http.NewRequest(method, fullURL, body)
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
		req, err = http.NewRequest(method, fullURL, nil)
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
		req, err = http.NewRequest(method, fullURL, strings.NewReader(values.Encode()))
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
		var errResp types.Response
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Msg != "" {
			return body, fmt.Errorf("API error: %s", errResp.Msg)
		}
		return body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
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
func (c *Client) PostWithFiles(endpoint string, params map[string]interface{}, files map[string]io.Reader) ([]byte, error) {
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
