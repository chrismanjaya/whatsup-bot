package group

import (
	"context"
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByIDQuery = `
	SELECT ` + groupColumns + ` FROM groups 
	WHERE 
		id = ?
`

func (r *Repository) FindByID(ctx context.Context, groupID int64) (*domain.Group, error) {
	row := r.db.QueryRowContext(ctx, findByIDQuery, groupID)
	result, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, util.Wrap(err, "query group failed")
	}

	return result, nil
}
