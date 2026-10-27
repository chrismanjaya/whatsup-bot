package transaction

import (
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const transactionColumns = `
	id,
	user_id,
	description,
	type, 
	amount, 
	category,
	is_shared, 
	group_id, 
	message_id,
	transaction_date,
	created_at,
	COUNT(*) OVER() AS total_count
`

func scanRow(row *sql.Row) (*domain.Transaction, int, error) {
	return scanRowFrom(row.Scan)
}

func scanRows(rows *sql.Rows) ([]*domain.Transaction, int, error) {
	var (
		results     []*domain.Transaction
		totalCount  int
		transaction *domain.Transaction
		err         error
	)

	for rows.Next() {
		transaction, totalCount, err = scanRowFrom(rows.Scan)
		if err != nil {
			return nil, 0, err
		}

		if transaction == nil {
			continue
		}

		results = append(results, transaction)
	}

	return results, totalCount, rows.Err()
}

func scanRowFrom(scan func(dest ...any) error) (*domain.Transaction, int, error) {
	var transaction domain.Transaction
	var transactionType string
	var totalCount int

	err := scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.Description,
		&transactionType,
		&transaction.Amount,
		&transaction.Category,
		&transaction.IsShared,
		&transaction.GroupID,
		&transaction.MessageID,
		&transaction.TransactionDate,
		&transaction.CreatedAt,
		&totalCount,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, nil
	}

	if err != nil {
		return nil, 0, util.Wrap(err, "scan transaction")
	}

	transaction.Type = domain.TransactionType(transactionType)
	return &transaction, totalCount, nil
}
