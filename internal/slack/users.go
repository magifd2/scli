package slack

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"
)

const userCacheTTL = time.Hour

// GetUser returns the User for the given Slack user ID.
// It checks the in-memory cache first, then the on-disk user list cache,
// and only falls back to an individual API call when necessary.
func (c *Client) GetUser(ctx context.Context, userID string) (User, error) {
	// 1. In-memory cache (populated by ListUsers or previous GetUser calls).
	if u, ok := c.userByID[userID]; ok {
		return u, nil
	}

	// 2. On-disk user list cache — load it into memory and retry.
	if c.cacheDir != "" {
		if users, ok := loadCache[[]User](filepath.Join(c.cacheDir, "users.json"), userCacheTTL); ok {
			c.indexUsers(users)
			if u, ok := c.userByID[userID]; ok {
				return u, nil
			}
		}
	}

	// 3. Individual API call for users not in any cache (e.g. bots, new members).
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

	u := User{
		ID:          resp.User.ID,
		Name:        resp.User.Name,
		DisplayName: resp.User.Profile.DisplayName,
		RealName:    resp.User.Profile.RealName,
		IsBot:       resp.User.IsBot,
		IsDeleted:   resp.User.Deleted,
	}
	if c.userByID == nil {
		c.userByID = make(map[string]User)
	}
	c.userByID[u.ID] = u
	return u, nil
}

// ListUsers returns all non-deleted, non-bot workspace members.
// Results are cached on disk for userCacheTTL to avoid repeated paginated fetches.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	if c.cacheDir != "" {
		if users, ok := loadCache[[]User](filepath.Join(c.cacheDir, "users.json"), userCacheTTL); ok {
			c.indexUsers(users)
			return users, nil
		}
	}

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

	if c.cacheDir != "" {
		saveCache(filepath.Join(c.cacheDir, "users.json"), all)
	}
	c.indexUsers(all)
	return all, nil
}

// indexUsers populates the in-memory userByID map from a slice of users.
func (c *Client) indexUsers(users []User) {
	if c.userByID == nil {
		c.userByID = make(map[string]User, len(users))
	}
	for _, u := range users {
		c.userByID[u.ID] = u
	}
}

// GetUserProfile returns the full profile for the given Slack user ID,
// including title, email, phone, status, and timezone.
func (c *Client) GetUserProfile(ctx context.Context, userID string) (UserProfile, error) {
	params := url.Values{"user": {userID}}

	var resp struct {
		User struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Deleted bool   `json:"deleted"`
			IsBot   bool   `json:"is_bot"`
			TZ      string `json:"tz"`
			TZLabel string `json:"tz_label"`
			Profile struct {
				DisplayName string `json:"display_name"`
				RealName    string `json:"real_name"`
				Title       string `json:"title"`
				Email       string `json:"email"`
				Phone       string `json:"phone"`
				StatusText  string `json:"status_text"`
				StatusEmoji string `json:"status_emoji"`
			} `json:"profile"`
		} `json:"user"`
	}

	if err := c.get(ctx, "users.info", params, &resp); err != nil {
		return UserProfile{}, fmt.Errorf("users.info: %w", err)
	}

	u := resp.User
	return UserProfile{
		User: User{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: u.Profile.DisplayName,
			RealName:    u.Profile.RealName,
			IsBot:       u.IsBot,
			IsDeleted:   u.Deleted,
		},
		Title:    u.Profile.Title,
		Email:    u.Profile.Email,
		Phone:    u.Profile.Phone,
		Status:   u.Profile.StatusText,
		Emoji:    u.Profile.StatusEmoji,
		Timezone: u.TZLabel,
	}, nil
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
