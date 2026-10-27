package user

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByJIDQuery = `
		SELECT ` + userColumns + ` FROM users 
		WHERE 
			jid = ?
	`

func (r *Repository) FindByJID(ctx context.Context, jid string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, findByJIDQuery, jid)
	result, err := scanRow(row)
	if err != nil {
		return nil, util.Wrap(err, "query user failed")
	}

	return result, nil
}
