package models

type Feed struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	UnreadCount int    `json:"unread_count"`
}

type Entry struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	Starred     bool   `json:"starred"`
	FeedID      int    `json:"feed_id"`
	FeedTitle   string `json:"feed_title"`
	PublishedAt string `json:"published_at"`
}
