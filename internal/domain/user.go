package domain

import "time"

type (
	User struct {
		ID         int64     `json:"id"`
		JID        string    `json:"jid"`
		Name       string    `json:"name"`
		Email      string    `json:"email"`
		RegisterAt time.Time `json:"register_at"`
	}
)
