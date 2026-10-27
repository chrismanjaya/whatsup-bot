package user

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByIDQuery = `
		SELECT ` + userColumns + ` FROM users 
		WHERE 
			id = ?
	`

func (r *Repository) FindByID(ctx context.Context, userID int64) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, findByIDQuery, userID)
	result, err := scanRow(row)
	if err != nil {
		return nil, util.Wrap(err, "query user failed")
	}

	return result, nil
}
