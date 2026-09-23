package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/utils"
)

type GroupRepo struct {
	db *sql.DB
}

func NewGroupRepo(db *sql.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

func (r *GroupRepo) Save(ctx context.Context, g *domain.Group) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO groups (jid, name, created_at) VALUES (?, ?, ?)`,
		g.JID, g.Name, g.CreatedAt,
	)
	if err != nil {
		return utils.Wrap(err, "insert group")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return utils.Wrap(err, "get last insert id")
	}
	g.ID = id
	return nil
}

func (r *GroupRepo) FindByJID(ctx context.Context, jid string) (*domain.Group, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, jid, name, created_at FROM groups WHERE jid = ?`, jid)
	var g domain.Group
	err := row.Scan(&g.ID, &g.JID, &g.Name, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, utils.Wrap(err, "query group")
	}
	return &g, nil
}

func (r *GroupRepo) FindByID(ctx context.Context, id int64) (*domain.Group, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, jid, name, created_at FROM groups WHERE id = ?`, id)
	var g domain.Group
	err := row.Scan(&g.ID, &g.JID, &g.Name, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, utils.Wrap(err, "query group")
	}
	return &g, nil
}

func (r *GroupRepo) AddMember(ctx context.Context, groupID, userID int64) error {
	// INSERT OR IGNORE makes this idempotent — re-adding an existing member is a no-op.
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?)`,
		groupID, userID,
	)
	if err != nil {
		return utils.Wrap(err, "insert group member")
	}
	return nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, groupID int64) ([]*domain.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.jid, u.name, u.email, u.register_at
		 FROM users u
		 JOIN group_members gm ON gm.user_id = u.id
		 WHERE gm.group_id = ?`,
		groupID,
	)
	if err != nil {
		return nil, utils.Wrap(err, "query members")
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.JID, &u.Name, &u.Email, &u.RegisterAt); err != nil {
			return nil, utils.Wrap(err, "scan member")
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
