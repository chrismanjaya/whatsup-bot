package example

import (
	"context"

	"example.com/app/internal/domain"
)

// ExampleRepository is what this package needs from storage, and nothing
// more. Declare ports where they are consumed and keep them narrow.
type ExampleRepository interface {
	Create(ctx context.Context, example *domain.Example) error
	FindByName(ctx context.Context, name string) (*domain.Example, error)
}
