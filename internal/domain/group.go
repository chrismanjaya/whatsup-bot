package domain

import "time"

type (
	Group struct {
		ID        int64     `json:"id"`
		JID       string    `json:"jid"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	GroupMember struct {
		GroupID int64 `json:"group_id"`
		UserID  int64 `json:"user_id"`
	}
)
