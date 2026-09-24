package domain

import "time"

// QueryPage is one page of a transaction listing the bot sent to a user. The
// summary message that ends the page is remembered by its WhatsApp message
// ID, so a reply to it ("next", "page 3") can be resolved back to the same
// date range without the user restating it.
type QueryPage struct {
	ID     int64
	UserID int64
	// From is inclusive and To is exclusive, both at Asia/Jakarta midnight.
	From        time.Time
	To          time.Time
	Page        int // 1-based
	WAMessageID string
	CreatedAt   time.Time
}
