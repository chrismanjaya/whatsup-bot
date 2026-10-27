package group

import (
	"context"
	"whatsupbot/internal/util"
)

const addMemberQuery = `
		INSERT OR IGNORE INTO group_members (
			group_id, 
			user_id
		) VALUES (?, ?)
	`

func (r *Repository) AddMember(ctx context.Context, groupID, userID int64) error {
	if _, err := r.db.ExecContext(ctx, addMemberQuery,
		groupID,
		userID,
	); err != nil {
		return util.Wrap(err, "insert group member")
	}

	return nil
}
