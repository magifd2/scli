package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	postFile   string
	postThread string
)

var postCmd = &cobra.Command{
	Use:   "post <channel> <message>",
	Short: "Post a message to a channel",
	Args:  cobra.ExactArgs(2),
	RunE:  runPost,
}

func init() {
	postCmd.Flags().StringVar(&postFile, "file", "", "Attach a file to the message")
	postCmd.Flags().StringVar(&postThread, "thread", "", "Reply in a thread (message timestamp)")
	rootCmd.AddCommand(postCmd)
}

func runPost(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]
	text := unescapeText(args[1])

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channelID, err := client.ResolveChannelID(cmd.Context(), nameOrID)
	if err != nil {
		return err
	}

	if postFile != "" {
		// File upload: message becomes the initial comment.
		if err := client.UploadFile(cmd.Context(), channelID, postFile, text); err != nil {
			return fmt.Errorf("upload file: %w", err)
		}
		p := newPrinter(cmd)
		p.Success("File uploaded.")
		return nil
	}

	var ts string
	if postThread != "" {
		ts, err = client.PostThreadReply(cmd.Context(), channelID, postThread, text)
	} else {
		ts, err = client.PostMessage(cmd.Context(), channelID, text)
	}
	if err != nil {
		return fmt.Errorf("post message: %w", err)
	}

	p := newPrinter(cmd)
	p.Success(fmt.Sprintf("Message posted (ts: %s)", ts))
	return nil
}
