package example

import (
	"context"

	"example.com/app/internal/domain"
	"example.com/app/internal/util"
)

const createQuery = `
	INSERT INTO examples (
		name,
		status
	) VALUES (?, ?)
`

func (r *Repository) Create(ctx context.Context, example *domain.Example) error {
	res, err := r.db.ExecContext(ctx, createQuery,
		example.Name,
		example.Status,
	)
	if err != nil {
		return util.Wrap(err, "insert example failed")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return util.Wrap(err, "get last insert id failed")
	}
	example.ID = id

	return nil
}
