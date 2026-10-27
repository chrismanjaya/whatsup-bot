package example

import (
	"context"
	"errors"
	"testing"

	"example.com/app/internal/constant"
	"example.com/app/internal/domain"
)

type fakeExampleRepo struct {
	byName    map[string]*domain.Example
	findErr   error
	createErr error
}

func (f *fakeExampleRepo) Create(_ context.Context, example *domain.Example) error {
	if f.createErr != nil {
		return f.createErr
	}
	example.ID = int64(len(f.byName) + 1)
	f.byName[example.Name] = example
	return nil
}

func (f *fakeExampleRepo) FindByName(_ context.Context, name string) (*domain.Example, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.byName[name], nil
}

func TestCreateExecute(t *testing.T) {
	dbErr := errors.New("db down")

	tests := []struct {
		name     string
		input    CreateInput
		existing map[string]*domain.Example
		findErr  error
		wantCode *constant.ErrCode
	}{
		{name: "creates example", input: CreateInput{Name: "alpha", Status: "active"}},
		{name: "rejects empty name", input: CreateInput{Name: "  ", Status: "active"}, wantCode: constant.ErrInvalidRequest},
		{name: "rejects unknown status", input: CreateInput{Name: "alpha", Status: "deleted"}, wantCode: constant.ErrExampleInvalidStatus},
		{
			name:     "rejects taken name",
			input:    CreateInput{Name: "alpha", Status: "active"},
			existing: map[string]*domain.Example{"alpha": {ID: 1, Name: "alpha"}},
			wantCode: constant.ErrExampleNameTaken,
		},
		{name: "classifies storage failure", input: CreateInput{Name: "alpha", Status: "active"}, findErr: dbErr, wantCode: constant.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeExampleRepo{byName: map[string]*domain.Example{}, findErr: tt.findErr}
			for k, v := range tt.existing {
				repo.byName[k] = v
			}

			got, err := NewCreate(repo).Execute(context.Background(), tt.input)

			if tt.wantCode != nil {
				if !errors.Is(err, tt.wantCode) {
					t.Fatalf("error = %v, want code %d", err, tt.wantCode.Code)
				}
				if tt.findErr != nil && !errors.Is(err, tt.findErr) {
					t.Fatalf("error = %v, want cause %v kept in chain", err, tt.findErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID == 0 || got.Name != tt.input.Name {
				t.Fatalf("got %+v, want stored example named %q", got, tt.input.Name)
			}
		})
	}
}
