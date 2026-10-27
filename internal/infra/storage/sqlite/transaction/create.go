package transaction

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const createQuery = `
		INSERT INTO transactions (
			user_id,
			description,
			type,
			amount,
			category,
			is_shared,
			group_id,
			transaction_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 
`

func (r *Repository) Create(ctx context.Context, transaction *domain.Transaction) error {
	res, err := r.db.ExecContext(ctx, createQuery,
		transaction.UserID,
		transaction.Description,
		transaction.Type,
		transaction.Amount,
		transaction.Category,
		transaction.IsShared,
		transaction.GroupID,
		transaction.TransactionDate,
	)
	if err != nil {
		return util.Wrap(err, "insert transaction failed")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return util.Wrap(err, "get last insert id failed")
	}
	transaction.ID = id
	return nil
}
