package group

type Module struct {
	validate *Validate
}

func NewModule(groupRepo GroupRepository, userRepo UserRepository) *Module {
	return &Module{
		validate: NewValidate(groupRepo, userRepo),
	}
}

func (m *Module) ValidateUsecase() *Validate {
	return m.validate
}
