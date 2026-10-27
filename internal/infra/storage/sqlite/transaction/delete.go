package transaction

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/util"
)

const deleteQuery = `DELETE FROM transactions WHERE id = ?`

func (r *Repository) Delete(ctx context.Context, transactionID int64) error {
	res, err := r.db.ExecContext(ctx, deleteQuery, transactionID)
	if err != nil {
		return util.Wrap(err, "delete transaction failed")
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
