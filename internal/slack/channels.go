package slack

import (
	"context"
	"fmt"
	"net/url"
)

// ListChannels returns all channels the authenticated user is a member of.
// It handles cursor-based pagination automatically.
func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	var all []Channel
	cursor := ""

	for {
		params := url.Values{
			"types":            {"public_channel,private_channel"},
			"exclude_archived": {"true"},
			"limit":            {"200"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Channels []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				IsMember   bool   `json:"is_member"`
				IsArchived bool   `json:"is_archived"`
				Purpose    struct {
					Value string `json:"value"`
				} `json:"purpose"`
				NumMembers  int `json:"num_members"`
				UnreadCount int `json:"unread_count"`
			} `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "conversations.list", params, &resp); err != nil {
			return nil, fmt.Errorf("conversations.list: %w", err)
		}

		for _, ch := range resp.Channels {
			if !ch.IsMember {
				continue
			}
			all = append(all, Channel{
				ID:          ch.ID,
				Name:        ch.Name,
				Purpose:     ch.Purpose.Value,
				IsMember:    ch.IsMember,
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

// GetChannelHistory returns up to limit messages from the channel identified
// by channelID, in chronological order (oldest first).
// If oldest is non-empty, only messages after that Slack timestamp are returned.
func (c *Client) GetChannelHistory(ctx context.Context, channelID string, limit int, oldest string) ([]Message, error) {
	params := url.Values{
		"channel": {channelID},
		"limit":   {fmt.Sprintf("%d", limit)},
	}
	if oldest != "" {
		params.Set("oldest", oldest)
		params.Set("inclusive", "false")
	}

	var resp struct {
		Messages []struct {
			TS         string `json:"ts"`
			User       string `json:"user"`
			BotID      string `json:"bot_id"`
			Username   string `json:"username"` // bot display name
			Text       string `json:"text"`
			ThreadTS   string `json:"thread_ts"`
			ReplyCount int    `json:"reply_count"`
			Files      []struct {
				ID                 string `json:"id"`
				Name               string `json:"name"`
				Mimetype           string `json:"mimetype"`
				URLPrivateDownload string `json:"url_private_download"`
			} `json:"files"`
		} `json:"messages"`
		HasMore bool `json:"has_more"`
	}

	if err := c.get(ctx, "conversations.history", params, &resp); err != nil {
		return nil, fmt.Errorf("conversations.history: %w", err)
	}

	// Slack returns newest-first; reverse to chronological order.
	msgs := make([]Message, len(resp.Messages))
	for i, m := range resp.Messages {
		files := make([]File, len(m.Files))
		for j, f := range m.Files {
			files[j] = File{
				ID:       f.ID,
				Name:     f.Name,
				MIMEType: f.Mimetype,
				URL:      f.URLPrivateDownload,
			}
		}
		msgs[len(resp.Messages)-1-i] = Message{
			Timestamp:   m.TS,
			UserID:      m.User,
			BotUsername: m.Username,
			Text:        m.Text,
			ThreadTS:    m.ThreadTS,
			ReplyCount:  m.ReplyCount,
			Files:       files,
		}
	}

	return msgs, nil
}

// ChannelInfo holds per-channel metadata returned by conversations.info.
type ChannelInfo struct {
	LastRead    string
	UnreadCount int
}

// GetChannelInfo returns metadata for the given channel, including last-read
// timestamp and unread message count.
func (c *Client) GetChannelInfo(ctx context.Context, channelID string) (ChannelInfo, error) {
	params := url.Values{"channel": {channelID}}

	var resp struct {
		Channel struct {
			LastRead    string `json:"last_read"`
			UnreadCount int    `json:"unread_count"`
		} `json:"channel"`
	}

	if err := c.get(ctx, "conversations.info", params, &resp); err != nil {
		return ChannelInfo{}, fmt.Errorf("conversations.info: %w", err)
	}
	return ChannelInfo{
		LastRead:    resp.Channel.LastRead,
		UnreadCount: resp.Channel.UnreadCount,
	}, nil
}

// GetChannelLastRead returns the last-read timestamp for the given channel.
// Returns an empty string if the information is unavailable.
func (c *Client) GetChannelLastRead(ctx context.Context, channelID string) (string, error) {
	info, err := c.GetChannelInfo(ctx, channelID)
	if err != nil {
		return "", err
	}
	return info.LastRead, nil
}

// PostMessage sends a text message to the given channel.
// Returns the message timestamp on success.
func (c *Client) PostMessage(ctx context.Context, channelID, text string) (string, error) {
	params := url.Values{
		"channel": {channelID},
		"text":    {text},
	}

	var resp struct {
		TS string `json:"ts"`
	}

	if err := c.post(ctx, "chat.postMessage", params, &resp); err != nil {
		return "", fmt.Errorf("chat.postMessage: %w", err)
	}
	return resp.TS, nil
}

// PostThreadReply sends a message as a reply to a thread.
func (c *Client) PostThreadReply(ctx context.Context, channelID, threadTS, text string) (string, error) {
	params := url.Values{
		"channel":   {channelID},
		"text":      {text},
		"thread_ts": {threadTS},
	}

	var resp struct {
		TS string `json:"ts"`
	}

	if err := c.post(ctx, "chat.postMessage", params, &resp); err != nil {
		return "", fmt.Errorf("chat.postMessage (thread): %w", err)
	}
	return resp.TS, nil
}
