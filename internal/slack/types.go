// Package slack provides a Slack Web API client for scli.
package slack

// Channel represents a Slack channel (public, private, or DM).
type Channel struct {
	ID          string
	Name        string
	Purpose     string
	IsIM        bool // true for direct messages
	IsMPIM      bool // true for group direct messages
	IsMember    bool
	UnreadCount int
	UserID      string // for DMs: the other user's Slack ID
}

// Message represents a single Slack message.
type Message struct {
	Timestamp   string // Slack timestamp (e.g. "1234567890.123456")
	UserID      string
	UserName    string // resolved display name (may be empty if unresolved)
	BotUsername string // set for bot messages that have no user ID
	Text        string
	ThreadTS    string // non-empty if this is a thread reply or thread parent
	ReplyCount  int    // number of replies (>0 means this is a thread parent)
	Files       []File
	Replies     []Message // populated when thread replies are fetched
}

// File represents a file attached to a message.
type File struct {
	ID       string
	Name     string
	MIMEType string
	URL      string
}

// SearchResult represents a single search match returned by search.messages.
type SearchResult struct {
	ChannelID   string
	ChannelName string
	Message     Message
}

// User represents a Slack workspace member.
type User struct {
	ID          string
	Name        string // username (handle)
	DisplayName string // profile display name
	RealName    string
	IsBot       bool
	IsDeleted   bool
}

// UserProfile extends User with detailed profile fields returned by users.info.
type UserProfile struct {
	User
	Title    string
	Email    string
	Phone    string
	Status   string // status text
	Emoji    string // status emoji
	Timezone string // tz label, e.g. "Asia/Tokyo"
}
