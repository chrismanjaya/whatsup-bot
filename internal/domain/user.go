package domain

import "time"

type (
	User struct {
		ID         int64
		JID        string
		Name       string
		Email      string
		RegisterAt time.Time
	}
)
