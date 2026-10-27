package transaction

import (
	"context"
	"time"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

var dateFormat = time.DateOnly

const (
	listTransactionQuery = `
		SELECT ` + transactionColumns + ` FROM transactions 
		WHERE 
			user_id = ?
			AND substr(transaction_date, 1, 10) >= ?
			AND substr(transaction_date, 1, 10) <= ?
		ORDER BY transaction_date ASC, id ASC 
		LIMIT ? 
		OFFSET ?
	`
)

func (r *Repository) GetListByPeriod(ctx context.Context, userID int64, from, to time.Time, limit, offset int) ([]*domain.Transaction, int, error) {
	fromStr, toStr := from.Format(dateFormat), to.Format(dateFormat)

	rows, err := r.db.QueryContext(ctx, listTransactionQuery,
		userID,
		fromStr,
		toStr,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, util.Wrap(err, "query transactions failed")
	}

	defer rows.Close()
	results, totalCount, err := scanRows(rows)
	if err != nil {
		return nil, 0, util.Wrap(err, "query transactions failed")
	}

	return results, totalCount, rows.Err()
}
