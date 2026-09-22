package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"whatsup-bot/internal/domain"
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
		return fmt.Errorf("insert group: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
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
		return nil, fmt.Errorf("query group: %w", err)
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
		return nil, fmt.Errorf("query group: %w", err)
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
		return fmt.Errorf("insert group member: %w", err)
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
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.JID, &u.Name, &u.Email, &u.RegisterAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
