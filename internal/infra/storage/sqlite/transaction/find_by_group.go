package transaction

import (
	"context"
	"time"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByGroupQuery = `
	SELECT ` + transactionColumns + ` FROM transactions 
	WHERE 
		group_id = ? 
		AND created_at >= ? 
	ORDER BY created_at DESC
`

func (r *Repository) FindByGroup(ctx context.Context, groupID int64, since time.Time) ([]*domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx, findByGroupQuery, groupID, since)
	if err != nil {
		return nil, util.Wrap(err, "query transactions failed")
	}
	defer rows.Close()
	results, _, err := scanRows(rows)
	return results, err
}
