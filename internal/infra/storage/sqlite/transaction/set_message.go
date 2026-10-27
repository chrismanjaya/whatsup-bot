package transaction

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/util"
)

const setMessageQuery = `
	UPDATE transactions SET 
		message_id = ? 
	WHERE id = ?
`

func (r *Repository) SetMessage(ctx context.Context, transactionID int64, messageID string) error {
	res, err := r.db.ExecContext(ctx, setMessageQuery, messageID, transactionID)
	if err != nil {
		return util.Wrap(err, "set message id failed")
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
