package transaction

import (
	"context"
	"database/sql"
	"time"
	"whatsupbot/internal/util"
)

const getLastTransactionTimeQuery = `
	SELECT created_at FROM transactions 
	WHERE 
		user_id = ? 
	ORDER BY id DESC LIMIT 1
`

func (r *Repository) GetLastTransactionTime(ctx context.Context, userID int64) (*time.Time, error) {
	var latestTime time.Time
	err := r.db.QueryRowContext(ctx, getLastTransactionTimeQuery, userID).Scan(&latestTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, util.Wrap(err, "last created_at")
	}
	return &latestTime, nil
}
