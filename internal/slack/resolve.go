package slack

import (
	"context"
	"fmt"
	"strings"
)

// ResolveChannelID resolves a channel name or ID to a Slack channel ID.
//
// Resolution rules:
//   - IDs starting with C, G, D (Slack channel ID prefixes) → used as-is
//   - Names starting with # → strip # and search by name
//   - Otherwise → search by name; error if ambiguous or not found
func (c *Client) ResolveChannelID(ctx context.Context, nameOrID string) (string, error) {
	// Already looks like a Slack channel ID
	if isChannelID(nameOrID) {
		return nameOrID, nil
	}

	name := strings.TrimPrefix(nameOrID, "#")
	name = strings.ToLower(name)

	channels, err := c.ListChannels(ctx)
	if err != nil {
		return "", fmt.Errorf("list channels for resolution: %w", err)
	}

	var matches []Channel
	for _, ch := range channels {
		if strings.ToLower(ch.Name) == name {
			matches = append(matches, ch)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("channel %q not found (use 'scli channel list' to see available channels)", nameOrID)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("channel name %q is ambiguous (%d matches); use a channel ID instead", nameOrID, len(matches))
	}
}

// ResolveUserID resolves a username or user ID to a Slack user ID.
//
// Resolution rules:
//   - IDs starting with U or W (Slack user ID prefixes) → used as-is
//   - Names starting with @ → strip @ and search by name
//   - Otherwise → search by name; error if ambiguous or not found
func (c *Client) ResolveUserID(ctx context.Context, nameOrID string) (string, error) {
	if isUserID(nameOrID) {
		return nameOrID, nil
	}

	name := strings.TrimPrefix(nameOrID, "@")
	name = strings.ToLower(name)

	users, err := c.ListUsers(ctx)
	if err != nil {
		return "", fmt.Errorf("list users for resolution: %w", err)
	}

	var matches []User
	for _, u := range users {
		if strings.ToLower(u.Name) == name ||
			strings.ToLower(u.DisplayName) == name ||
			strings.ToLower(u.RealName) == name {
			matches = append(matches, u)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("user %q not found (use 'scli user list' to see available users)", nameOrID)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("user name %q is ambiguous (%d matches); use a user ID instead", nameOrID, len(matches))
	}
}

// isChannelID reports whether s looks like a Slack channel/conversation ID.
func isChannelID(s string) bool {
	if len(s) < 2 {
		return false
	}
	switch s[0] {
	case 'C', 'G', 'D': // public channel, private channel/group, DM
		return true
	}
	return false
}

// isUserID reports whether s looks like a Slack user ID.
func isUserID(s string) bool {
	if len(s) < 2 {
		return false
	}
	switch s[0] {
	case 'U', 'W':
		return true
	}
	return false
}
