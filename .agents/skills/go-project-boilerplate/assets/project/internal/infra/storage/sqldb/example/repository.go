// Package example implements usecase/example.ExampleRepository on
// database/sql. One file per method, each holding its own query constant.
package example

import "database/sql"

type Repository struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repository {
	return &Repository{db: db}
}
