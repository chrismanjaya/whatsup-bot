// Package domain holds entities and the rules that need no I/O. It is the
// center of the onion and imports nothing from this module.
package domain

import (
	"fmt"
	"slices"
	"time"
)

// ExampleStatus is a typed string enum: String/Valid/Parse live next to it.
type ExampleStatus string

const (
	ExampleStatusActive   ExampleStatus = "active"
	ExampleStatusArchived ExampleStatus = "archived"
)

// AllExampleStatuses is the single source of truth for valid statuses.
var AllExampleStatuses = []ExampleStatus{
	ExampleStatusActive,
	ExampleStatusArchived,
}

// Example is a sample entity. Replace it with your own.
type Example struct {
	ID        int64
	Name      string
	Status    ExampleStatus
	CreatedAt time.Time
}

func (s ExampleStatus) String() string { return string(s) }

// Valid reports whether s is one of the defined values.
func (s ExampleStatus) Valid() bool { return slices.Contains(AllExampleStatuses, s) }

// ParseExampleStatus converts an external string (request body, DB row, LLM
// output) into an ExampleStatus, rejecting unknown values.
func ParseExampleStatus(s string) (ExampleStatus, error) {
	status := ExampleStatus(s)
	if !status.Valid() {
		return "", fmt.Errorf("invalid example status: %q", s)
	}
	return status, nil
}
