package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultBaseURL = "https://slack.com/api"

// Client is a Slack Web API client authenticated as a user.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
	cacheDir   string          // optional; empty = no disk cache
	userByID   map[string]User // in-memory user cache, keyed by Slack user ID
}

// NewClient returns a Client authenticated with the given user token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// SetBaseURL overrides the Slack API base URL. Intended for tests.
func (c *Client) SetBaseURL(u string) {
	c.baseURL = u
}

// SetCacheDir enables disk caching of slow list endpoints (channels, users).
// dir is workspace-specific and is created on first use.
func (c *Client) SetCacheDir(dir string) {
	c.cacheDir = dir
}

// apiResponse is the common envelope returned by every Slack Web API call.
type apiResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// get calls a Slack API method with URL-encoded query parameters and decodes
// the JSON response into dst.
func (c *Client) get(ctx context.Context, method string, params url.Values, dst interface{}) error {
	rawURL := c.baseURL + "/" + method + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	return c.do(req, dst)
}

// post calls a Slack API method with URL-encoded form parameters and decodes
// the JSON response into dst.
func (c *Client) post(ctx context.Context, method string, params url.Values, dst interface{}) error {
	rawURL := c.baseURL + "/" + method

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.do(req, dst)
}

// do executes req, retrying up to maxRetries times on HTTP 429 responses.
// It honours the Retry-After header returned by Slack and cancels early if
// the request context is done.
func (c *Client) do(req *http.Request, dst interface{}) error {
	const maxRetries = 3

	for attempt := range maxRetries {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := retryAfterDuration(resp.Header.Get("Retry-After"))
			resp.Body.Close() //nolint:errcheck
			if attempt == maxRetries-1 {
				break
			}
			select {
			case <-time.After(wait):
			case <-req.Context().Done():
				return req.Context().Err()
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close() //nolint:errcheck
			return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}

		// Check the Slack-level ok/error fields first.
		var base apiResponse
		if err := json.Unmarshal(body, &base); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !base.OK {
			return fmt.Errorf("slack API error: %s", base.Error)
		}

		if err := json.Unmarshal(body, dst); err != nil {
			return fmt.Errorf("decode response payload: %w", err)
		}
		return nil
	}

	return fmt.Errorf("rate limited by Slack (HTTP 429): still throttled after %d attempts", maxRetries)
}

// retryAfterDuration parses the Retry-After header value (seconds) and returns
// the corresponding duration plus a one-second buffer. Falls back to 5 seconds.
func retryAfterDuration(header string) time.Duration {
	if n, err := strconv.Atoi(header); err == nil && n > 0 {
		return time.Duration(n+1) * time.Second
	}
	return 5 * time.Second
}

// RetryAfterDurationForTest exposes retryAfterDuration for use in tests.
func RetryAfterDurationForTest(header string) time.Duration {
	return retryAfterDuration(header)
}
