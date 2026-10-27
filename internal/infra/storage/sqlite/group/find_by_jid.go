package group

import (
	"context"
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByJIDQuery = `
	SELECT ` + groupColumns + ` FROM groups 
	WHERE 
		jid = ?
`

func (r *Repository) FindByJID(ctx context.Context, jid string) (*domain.Group, error) {
	row := r.db.QueryRowContext(ctx, groupColumns, jid)
	result, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, util.Wrap(err, "query group failed")
	}
	return result, nil
}
