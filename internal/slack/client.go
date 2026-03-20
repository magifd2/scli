package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://slack.com/api"

// Client is a Slack Web API client authenticated as a user.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
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

// apiResponse is the common envelope returned by every Slack Web API call.
type apiResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// get calls a Slack API method with URL-encoded query parameters and decodes
// the JSON response into dst.
func (c *Client) get(ctx context.Context, method string, params url.Values, dst interface{}) error {
	params.Set("limit", params.Get("limit")) // keep caller-supplied limit
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

func (c *Client) do(req *http.Request, dst interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited by Slack (HTTP 429) — please wait and retry")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
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
