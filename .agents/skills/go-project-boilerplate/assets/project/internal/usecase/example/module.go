// Package example holds the example feature's usecases: one struct per
// business operation, each with a single Execute method.
package example

// Module builds every usecase of this package from the same dependencies.
type Module struct {
	create *Create
}

func NewModule(exampleRepo ExampleRepository) *Module {
	return &Module{
		create: NewCreate(exampleRepo),
	}
}

func (m *Module) CreateUsecase() *Create {
	return m.create
}
