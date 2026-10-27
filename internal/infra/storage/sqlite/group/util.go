package group

import (
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const groupColumns = `
	id, 
	jid, 
	name, 
	created_at
`

func scanRow(row *sql.Row) (*domain.Group, error) {
	return scanRowFrom(row.Scan)
}

func scanRows(rows *sql.Rows) ([]*domain.Group, error) {
	var results []*domain.Group
	for rows.Next() {
		group, err := scanRowFrom(rows.Scan)
		if err != nil {
			return nil, err
		}

		if group == nil {
			continue
		}

		results = append(results, group)
	}
	return results, rows.Err()
}

func scanRowFrom(scan func(dest ...any) error) (*domain.Group, error) {
	var group domain.Group

	err := scan(
		&group.ID,
		&group.JID,
		&group.Name,
		&group.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, util.Wrap(err, "scan group")
	}

	return &group, nil
}
