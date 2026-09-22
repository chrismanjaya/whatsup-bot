package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"whatsup-bot/internal/domain"
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
		return fmt.Errorf("insert transaction failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id failed: %w", err)
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
		return nil, fmt.Errorf("query transactions: %w", err)
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
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func scanTransactions(rows *sql.Rows) ([]*domain.Transaction, error) {
	var results []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		var txType string
		var groupID sql.NullInt64

		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Description, &txType, &tx.Amount, &tx.Category, &tx.IsShared, &groupID, &tx.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
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
