package slack

import (
	"context"
	"fmt"
	"net/url"
)

// OpenDM opens (or retrieves the existing) DM channel with the given user.
// Returns the DM channel ID (D-prefixed).
func (c *Client) OpenDM(ctx context.Context, userID string) (string, error) {
	params := url.Values{"users": {userID}}

	var resp struct {
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}

	if err := c.post(ctx, "conversations.open", params, &resp); err != nil {
		return "", fmt.Errorf("conversations.open: %w", err)
	}
	return resp.Channel.ID, nil
}

// ListDMs returns all open 1:1 DM conversations.
// Each Channel has IsIM=true and UserID set to the other user's ID.
// It handles cursor-based pagination automatically.
func (c *Client) ListDMs(ctx context.Context) ([]Channel, error) {
	var all []Channel
	cursor := ""

	for {
		params := url.Values{
			"types": {"im"},
			"limit": {"200"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Channels []struct {
				ID          string `json:"id"`
				User        string `json:"user"`
				UnreadCount int    `json:"unread_count"`
			} `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "conversations.list", params, &resp); err != nil {
			return nil, fmt.Errorf("conversations.list (im): %w", err)
		}

		for _, ch := range resp.Channels {
			all = append(all, Channel{
				ID:          ch.ID,
				IsIM:        true,
				UserID:      ch.User,
				UnreadCount: ch.UnreadCount,
			})
		}

		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	return all, nil
}
