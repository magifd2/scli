package slack

import (
	"context"
	"fmt"
	"net/url"
)

// GetUser returns the User for the given Slack user ID.
func (c *Client) GetUser(ctx context.Context, userID string) (User, error) {
	params := url.Values{"user": {userID}}

	var resp struct {
		User struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Deleted bool   `json:"deleted"`
			IsBot   bool   `json:"is_bot"`
			Profile struct {
				DisplayName string `json:"display_name"`
				RealName    string `json:"real_name"`
			} `json:"profile"`
		} `json:"user"`
	}

	if err := c.get(ctx, "users.info", params, &resp); err != nil {
		return User{}, fmt.Errorf("users.info: %w", err)
	}

	u := resp.User
	return User{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: u.Profile.DisplayName,
		RealName:    u.Profile.RealName,
		IsBot:       u.IsBot,
		IsDeleted:   u.Deleted,
	}, nil
}

// ListUsers returns all non-deleted, non-bot workspace members.
// It handles cursor-based pagination automatically.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var all []User
	cursor := ""

	for {
		params := url.Values{"limit": {"200"}}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Members []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Deleted bool   `json:"deleted"`
				IsBot   bool   `json:"is_bot"`
				Profile struct {
					DisplayName string `json:"display_name"`
					RealName    string `json:"real_name"`
				} `json:"profile"`
			} `json:"members"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "users.list", params, &resp); err != nil {
			return nil, fmt.Errorf("users.list: %w", err)
		}

		for _, m := range resp.Members {
			if m.Deleted || m.IsBot {
				continue
			}
			all = append(all, User{
				ID:          m.ID,
				Name:        m.Name,
				DisplayName: m.Profile.DisplayName,
				RealName:    m.Profile.RealName,
			})
		}

		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	return all, nil
}

// ResolveUserName returns a human-readable name for the given user ID.
// It tries DisplayName first, then RealName, then the handle.
// Falls back to the raw ID if the API call fails.
func (c *Client) ResolveUserName(ctx context.Context, userID string) string {
	if userID == "" {
		return ""
	}
	u, err := c.GetUser(ctx, userID)
	if err != nil {
		return userID // graceful degradation
	}
	return displayName(u)
}

// displayName returns the best available display name for a User.
func displayName(u User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.RealName != "" {
		return u.RealName
	}
	return u.Name
}
