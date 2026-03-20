package slack

import (
	"context"
	"fmt"
	"net/url"
)

// GetThreadReplies returns up to limit messages from a thread, in chronological
// order (oldest first), including the parent message.
func (c *Client) GetThreadReplies(ctx context.Context, channelID, threadTS string, limit int) ([]Message, error) {
	params := url.Values{
		"channel": {channelID},
		"ts":      {threadTS},
		"limit":   {fmt.Sprintf("%d", limit)},
	}

	var resp struct {
		Messages []struct {
			TS       string `json:"ts"`
			User     string `json:"user"`
			BotID    string `json:"bot_id"`
			Username string `json:"username"`
			Text     string `json:"text"`
			ThreadTS string `json:"thread_ts"`
			Files    []struct {
				ID                 string `json:"id"`
				Name               string `json:"name"`
				Mimetype           string `json:"mimetype"`
				URLPrivateDownload string `json:"url_private_download"`
			} `json:"files"`
		} `json:"messages"`
		HasMore bool `json:"has_more"`
	}

	if err := c.get(ctx, "conversations.replies", params, &resp); err != nil {
		return nil, fmt.Errorf("conversations.replies: %w", err)
	}

	// conversations.replies returns messages in chronological order already.
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
		msgs[i] = Message{
			Timestamp:   m.TS,
			UserID:      m.User,
			BotUsername: m.Username,
			Text:        m.Text,
			ThreadTS:    m.ThreadTS,
			Files:       files,
		}
	}

	return msgs, nil
}
