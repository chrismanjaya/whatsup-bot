package example

import (
	"database/sql"
	"errors"

	"example.com/app/internal/domain"
	"example.com/app/internal/util"
)

const exampleColumns = `
	id,
	name,
	status,
	created_at
`

// scanRowFrom scans one row in exampleColumns order. It accepts row.Scan or
// rows.Scan, so single-row and multi-row queries share one mapping.
func scanRowFrom(scan func(dest ...any) error) (*domain.Example, error) {
	var (
		example domain.Example
		status  string
	)

	err := scan(
		&example.ID,
		&example.Name,
		&status,
		&example.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, util.Wrap(err, "scan example failed")
	}

	example.Status, err = domain.ParseExampleStatus(status)
	if err != nil {
		return nil, util.Wrap(err, "map example status failed")
	}

	return &example, nil
}
