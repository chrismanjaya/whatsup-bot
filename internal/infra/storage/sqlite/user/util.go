package user

import (
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const userColumns = `
	id, 
	jid, 
	name, 
	email, 
	register_at 
`

func scanRow(row *sql.Row) (*domain.User, error) {
	return scanRowFrom(row.Scan)
}

func scanRows(rows *sql.Rows) ([]*domain.User, error) {
	var results []*domain.User
	for rows.Next() {
		user, err := scanRowFrom(rows.Scan)
		if err != nil {
			return nil, err
		}

		if user == nil {
			continue
		}

		results = append(results, user)
	}

	return results, rows.Err()
}

func scanRowFrom(scan func(dest ...any) error) (*domain.User, error) {
	var user domain.User

	err := scan(
		&user.ID,
		&user.JID,
		&user.Name,
		&user.Email,
		&user.RegisterAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, util.Wrap(err, "scan user")
	}

	return &user, nil
}
