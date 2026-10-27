package transaction

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByMessageQuery = `
	SELECT ` + transactionColumns + ` 
	FROM transactions 
	WHERE 
		message_id = ?
`

func (r *Repository) FindByMessage(ctx context.Context, messageID string) (*domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx, findByMessageQuery, messageID)
	result, _, err := scanRow(row)
	if err != nil {
		return nil, util.Wrap(err, "query transactions failed")
	}

	return result, nil
}
