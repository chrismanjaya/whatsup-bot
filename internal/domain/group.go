package domain

import "time"

type (
	Group struct {
		ID        int64
		JID       string
		Name      string
		CreatedAt time.Time
	}

	GroupMember struct {
		GroupID int64
		UserID  int64
	}
)
