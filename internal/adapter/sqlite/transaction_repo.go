package sqlite

import (
	"context"
	"database/sql"
	"time"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/utils"
)

type TransactionRepo struct {
	db *sql.DB
}

func NewTransactionRepo(db *sql.DB) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) Save(ctx context.Context, tx *domain.Transaction) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO transactions (user_id, description, type, amount, category, is_shared, group_id, transaction_date, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.UserID, tx.Description, tx.Type, tx.Amount, tx.Category, tx.IsShared, nullableGroupID(tx.GroupID), tx.TransactionDate, tx.CreatedAt,
	)
	if err != nil {
		return utils.Wrap(err, "insert transaction failed")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return utils.Wrap(err, "get last insert id failed")
	}
	tx.ID = id
	return nil
}

func (r *TransactionRepo) FindByUser(ctx context.Context, userID int64, since time.Time) ([]*domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, description, type, amount, category, is_shared, group_id, created_at
		 FROM transactions WHERE user_id = ? AND created_at >= ? ORDER BY created_at DESC`,
		userID, since,
	)
	if err != nil {
		return nil, utils.Wrap(err, "query transactions")
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *TransactionRepo) FindByGroup(ctx context.Context, groupID int64, since time.Time) ([]*domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, description, type, amount, category, is_shared, group_id, created_at
		 FROM transactions WHERE group_id = ? AND created_at >= ? ORDER BY created_at DESC`,
		groupID, since,
	)
	if err != nil {
		return nil, utils.Wrap(err, "query transactions")
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *TransactionRepo) FindByWAMessageID(ctx context.Context, waMessageID string) (*domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, description, type, amount, category, is_shared, group_id, transaction_date, created_at
		 FROM transactions WHERE wa_message_id = ?`,
		waMessageID,
	)

	var tx domain.Transaction
	var txType string
	var groupID sql.NullInt64
	if err := row.Scan(&tx.ID, &tx.UserID, &tx.Description, &txType, &tx.Amount, &tx.Category, &tx.IsShared, &groupID, &tx.TransactionDate, &tx.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.Wrap(err, "find transaction by wa message id")
	}
	tx.Type = domain.TransactionType(txType)
	if groupID.Valid {
		tx.GroupID = groupID.Int64
	}
	tx.WAMessageID = waMessageID
	return &tx, nil
}

func (r *TransactionRepo) SetWAMessageID(ctx context.Context, id int64, waMessageID string) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE transactions SET wa_message_id = ? WHERE id = ?`, waMessageID, id); err != nil {
		return utils.Wrap(err, "set wa message id failed")
	}
	return nil
}

func (r *TransactionRepo) Update(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE transactions SET description = ?, type = ?, amount = ?, category = ?, transaction_date = ? WHERE id = ?`,
		tx.Description, tx.Type, tx.Amount, tx.Category, tx.TransactionDate, tx.ID,
	)
	if err != nil {
		return utils.Wrap(err, "update transaction failed")
	}
	return nil
}

func (r *TransactionRepo) Delete(ctx context.Context, id int64) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id); err != nil {
		return utils.Wrap(err, "delete transaction failed")
	}
	return nil
}

func scanTransactions(rows *sql.Rows) ([]*domain.Transaction, error) {
	var results []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		var txType string
		var groupID sql.NullInt64

		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Description, &txType, &tx.Amount, &tx.Category, &tx.IsShared, &groupID, &tx.CreatedAt); err != nil {
			return nil, utils.Wrap(err, "scan transaction")
		}
		tx.Type = domain.TransactionType(txType)
		if groupID.Valid {
			tx.GroupID = groupID.Int64
		}
		results = append(results, &tx)
	}
	return results, rows.Err()
}

// nullableGroupID treats a zero GroupID as "no group" (personal transaction),
// storing NULL instead of the literal 0 — safe since autoincrement IDs start at 1.
func nullableGroupID(id int64) interface{} {
	if id == 0 {
		return nil
	}
	return id
}

// ListByUserBetween compares on the stored local date (the first 10 chars of
// transaction_date, "YYYY-MM-DD") rather than the full timestamp, since
// timestamps are stored as text with a UTC offset and only compare correctly
// when the offsets match.
func (r *TransactionRepo) ListByUserBetween(ctx context.Context, userID int64, from, to time.Time, limit, offset int) ([]*domain.Transaction, int, error) {
	const dateFmt = "2006-01-02"
	fromStr, toStr := from.Format(dateFmt), to.Format(dateFmt)

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM transactions
		 WHERE user_id = ? AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) < ?`,
		userID, fromStr, toStr,
	).Scan(&total); err != nil {
		return nil, 0, utils.Wrap(err, "count transactions")
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, description, type, amount, category, is_shared, group_id, transaction_date, created_at
		 FROM transactions
		 WHERE user_id = ? AND substr(transaction_date, 1, 10) >= ? AND substr(transaction_date, 1, 10) < ?
		 ORDER BY transaction_date ASC, id ASC LIMIT ? OFFSET ?`,
		userID, fromStr, toStr, limit, offset,
	)
	if err != nil {
		return nil, 0, utils.Wrap(err, "query transactions")
	}
	defer rows.Close()

	var results []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		var txType string
		var groupID sql.NullInt64
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Description, &txType, &tx.Amount, &tx.Category, &tx.IsShared, &groupID, &tx.TransactionDate, &tx.CreatedAt); err != nil {
			return nil, 0, utils.Wrap(err, "scan transaction")
		}
		tx.Type = domain.TransactionType(txType)
		if groupID.Valid {
			tx.GroupID = groupID.Int64
		}
		results = append(results, &tx)
	}
	return results, total, rows.Err()
}
