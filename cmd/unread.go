package cmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/magifd2/scli/internal/slack"
	"github.com/spf13/cobra"
)

var unreadCmd = &cobra.Command{
	Use:   "unread",
	Short: "Show channels and DMs with unread messages",
	RunE:  runUnread,
}

func init() {
	rootCmd.AddCommand(unreadCmd)
}

const unreadConcurrency = 10

func runUnread(cmd *cobra.Command, _ []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channels, err := client.ListChannels(cmd.Context())
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	dms, err := client.ListDMs(cmd.Context())
	if err != nil {
		return fmt.Errorf("list DMs: %w", err)
	}

	// Fetch unread counts via conversations.info in parallel.
	populateUnreadCounts(cmd.Context(), client, channels)
	populateUnreadCounts(cmd.Context(), client, dms)

	var unreadChannels []slack.Channel
	for _, ch := range channels {
		if ch.UnreadCount > 0 {
			unreadChannels = append(unreadChannels, ch)
		}
	}

	seen := map[string]string{}
	var unreadDMs []slack.Channel
	for _, ch := range dms {
		if ch.UnreadCount == 0 {
			continue
		}
		if ch.UserID != "" {
			if name, ok := seen[ch.UserID]; ok {
				ch.Name = name
			} else {
				name := client.ResolveUserName(cmd.Context(), ch.UserID)
				seen[ch.UserID] = name
				ch.Name = name
			}
		}
		unreadDMs = append(unreadDMs, ch)
	}

	p := newPrinter(cmd)
	return p.Unread(unreadChannels, unreadDMs)
}

// populateUnreadCounts determines whether each channel has unread messages,
// using up to unreadConcurrency parallel requests.
//
// Strategy:
//  1. conversations.info → unread_count (fast path; may be 0 for bot/webhook posts)
//  2. If unread_count == 0 and last_read is known, fall back to conversations.history
//     with oldest=last_read to detect messages that the API did not count.
func populateUnreadCounts(ctx context.Context, client *slack.Client, channels []slack.Channel) {
	sem := make(chan struct{}, unreadConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range channels {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info, err := client.GetChannelInfo(ctx, channels[i].ID)
			if err != nil {
				return
			}

			count := info.UnreadCount
			if count == 0 && info.LastRead != "" {
				// conversations.info may under-report for bot/webhook messages.
				// Check history directly.
				msgs, err := client.GetChannelHistory(ctx, channels[i].ID, 1, info.LastRead)
				if err == nil && len(msgs) > 0 {
					count = len(msgs) // at least 1 unread
				}
			}

			mu.Lock()
			channels[i].UnreadCount = count
			mu.Unlock()
		}(i)
	}
	wg.Wait()
}
