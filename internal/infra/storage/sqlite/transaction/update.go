package transaction

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const updateQuery = `
	UPDATE transactions SET 
		description = ?, 
		type = ?, 
		amount = ?, 
		category = ?, 
		transaction_date = ? 
	WHERE id = ?
`

func (r *Repository) Update(ctx context.Context, transaction *domain.Transaction) error {
	res, err := r.db.ExecContext(ctx, updateQuery,
		transaction.Description,
		transaction.Type,
		transaction.Amount,
		transaction.Category,
		transaction.TransactionDate,
		transaction.ID,
	)
	if err != nil {
		return util.Wrap(err, "update transaction failed")
	}
	count, err := res.RowsAffected()
	if err != nil {
		return util.Wrap(err, "rows affected")
	}
	if count == 0 {
		return util.Wrap(constant.ErrTransactionNotFound, "transaction not found")
	}

	return nil
}
