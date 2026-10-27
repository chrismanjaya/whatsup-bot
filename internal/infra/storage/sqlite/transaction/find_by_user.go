package transaction

import (
	"context"
	"time"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByUserQuery = `
	SELECT ` + transactionColumns + ` FROM transactions 
	WHERE 
		user_id = ? 
		AND created_at >= ? 
	ORDER BY created_at DESC
`

func (r *Repository) FindByUser(ctx context.Context, userID int64, since time.Time) ([]*domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx, findByUserQuery, userID, since)
	if err != nil {
		return nil, util.Wrap(err, "query transactions failed")
	}
	defer rows.Close()
	results, _, err := scanRows(rows)
	return results, err
}
