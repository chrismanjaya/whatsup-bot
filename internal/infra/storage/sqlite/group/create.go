package group

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const createQuery = `
	INSERT INTO groups (
		jid, 
		name
	) VALUES (?, ?)
`

func (r *Repository) Create(ctx context.Context, group *domain.Group) error {
	res, err := r.db.ExecContext(ctx, createQuery,
		group.JID,
		group.Name,
	)
	if err != nil {
		return util.Wrap(err, "insert group")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return util.Wrap(err, "get last insert id")
	}
	group.ID = id
	return nil
}
