package example

import (
	"context"

	"example.com/app/internal/domain"
	"example.com/app/internal/util"
)

const findByNameQuery = `
	SELECT ` + exampleColumns + ` FROM examples
	WHERE
		name = ?
`

// FindByName returns (nil, nil) when no row matches: "not found" is a normal
// outcome the usecase decides about, not a storage error.
func (r *Repository) FindByName(ctx context.Context, name string) (*domain.Example, error) {
	row := r.db.QueryRowContext(ctx, findByNameQuery, name)
	result, err := scanRowFrom(row.Scan)
	if err != nil {
		return nil, util.Wrap(err, "query example failed")
	}

	return result, nil
}
