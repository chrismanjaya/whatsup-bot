package user

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const createQuery = `
		INSERT INTO users (
			jid, 
			name,
			email
		) VALUES (?, ?, ?)
`

func (r *Repository) Create(ctx context.Context, u *domain.User) error {
	res, err := r.db.ExecContext(ctx, createQuery,
		u.JID,
		u.Name,
		u.Email,
	)
	if err != nil {
		return util.Wrap(err, "insert user failed")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return util.Wrap(err, "get last insert id failed")
	}
	u.ID = id
	return nil
}
