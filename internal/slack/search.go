package slack

import (
	"context"
	"fmt"
	"net/url"
)

// SearchMessages searches workspace messages matching query.
// sortAsc=true returns results in chronological order (oldest first);
// sortAsc=false (default) returns newest first.
// Returns up to count results (max 100 per page; only the first page is fetched).
func (c *Client) SearchMessages(ctx context.Context, query string, count int, sortAsc bool) ([]SearchResult, error) {
	if count > 100 {
		count = 100
	}
	sortDir := "desc"
	if sortAsc {
		sortDir = "asc"
	}
	params := url.Values{
		"query":    {query},
		"count":    {fmt.Sprintf("%d", count)},
		"sort":     {"timestamp"},
		"sort_dir": {sortDir},
	}

	var resp struct {
		Messages struct {
			Matches []struct {
				Channel struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"channel"`
				TS       string `json:"ts"`
				ThreadTS string `json:"thread_ts"`
				User     string `json:"user"`
				Username string `json:"username"`
				Text     string `json:"text"`
				Files    []struct {
					ID                 string `json:"id"`
					Name               string `json:"name"`
					Mimetype           string `json:"mimetype"`
					URLPrivateDownload string `json:"url_private_download"`
				} `json:"files"`
			} `json:"matches"`
			Total int `json:"total"`
		} `json:"messages"`
	}

	if err := c.get(ctx, "search.messages", params, &resp); err != nil {
		return nil, fmt.Errorf("search.messages: %w", err)
	}

	results := make([]SearchResult, len(resp.Messages.Matches))
	for i, m := range resp.Messages.Matches {
		files := make([]File, len(m.Files))
		for j, f := range m.Files {
			files[j] = File{
				ID:       f.ID,
				Name:     f.Name,
				MIMEType: f.Mimetype,
				URL:      f.URLPrivateDownload,
			}
		}
		results[i] = SearchResult{
			ChannelID:   m.Channel.ID,
			ChannelName: m.Channel.Name,
			Message: Message{
				Timestamp:   m.TS,
				ThreadTS:    m.ThreadTS,
				UserID:      m.User,
				BotUsername: m.Username,
				Text:        m.Text,
				Files:       files,
			},
		}
	}

	return results, nil
}
