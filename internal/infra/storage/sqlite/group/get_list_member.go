package group

import (
	"context"
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const getListMemberQuery = `
		SELECT 
			u.id, 
			u.jid, 
			u.name, 
			u.email, 
			u.register_at
		FROM users u
		JOIN group_members gm 
		ON 
			gm.user_id = u.id
		WHERE 
			gm.group_id = ?
	`

func (r *Repository) GetListMember(ctx context.Context, groupID int64) ([]*domain.User, error) {
	rows, err := r.db.QueryContext(ctx, getListMemberQuery, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, util.Wrap(err, "query members")
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.JID,
			&u.Name,
			&u.Email,
			&u.RegisterAt,
		); err != nil {
			return nil, util.Wrap(err, "scan member")
		}
		users = append(users, &u)
	}

	return users, rows.Err()
}
