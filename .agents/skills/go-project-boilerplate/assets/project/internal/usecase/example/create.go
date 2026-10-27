package example

import (
	"context"
	"strings"

	"example.com/app/internal/constant"
	"example.com/app/internal/domain"
	"example.com/app/internal/util"
)

const maxNameLength = 64

// Create registers a new example under a unique name.
type Create struct {
	exampleRepo ExampleRepository
}

func NewCreate(exampleRepo ExampleRepository) *Create {
	return &Create{exampleRepo: exampleRepo}
}

// CreateInput is the usecase's request. Delivery adapters map their own
// payloads onto it; the usecase never sees HTTP or chat types.
type CreateInput struct {
	Name   string
	Status string
}

func (uc *Create) Execute(ctx context.Context, input CreateInput) (*domain.Example, error) {
	// validate input
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > maxNameLength {
		return nil, util.Wrap(constant.ErrInvalidRequest, "name is empty or too long")
	}

	status, err := domain.ParseExampleStatus(input.Status)
	if err != nil {
		return nil, util.WrapStd(constant.ErrExampleInvalidStatus, "parse status failed", err)
	}

	// check uniqueness
	existing, err := uc.exampleRepo.FindByName(ctx, name)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "lookup example failed", err)
	}
	if existing != nil {
		return nil, util.Wrap(constant.ErrExampleNameTaken, "example name taken")
	}

	// create example
	example := &domain.Example{Name: name, Status: status}
	if err := uc.exampleRepo.Create(ctx, example); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "create example failed", err)
	}

	return example, nil
}
