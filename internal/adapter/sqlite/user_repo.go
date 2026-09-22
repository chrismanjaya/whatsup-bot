package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"whatsup-bot/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Save(ctx context.Context, u *domain.User) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (jid, name, email, register_at) VALUES (?, ?, ?, ?)`,
		u.JID, u.Name, u.Email, u.RegisterAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	u.ID = id
	return nil
}

func (r *UserRepo) FindByJID(ctx context.Context, jid string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, jid, name, email, register_at FROM users WHERE jid = ?`, jid,
	)
	var u domain.User
	err := row.Scan(&u.ID, &u.JID, &u.Name, &u.Email, &u.RegisterAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, jid, name, email, register_at FROM users WHERE id = ?`, id,
	)
	var u domain.User
	err := row.Scan(&u.ID, &u.JID, &u.Name, &u.Email, &u.RegisterAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &u, nil
}
