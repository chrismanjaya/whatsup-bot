package user

type Module struct {
	register *Register
	validate *Validate
}

func NewModule(userRepo UserRepository) *Module {
	return &Module{
		register: NewRegister(userRepo),
		validate: NewValidate(userRepo),
	}
}

func (m *Module) RegisterUsecase() *Register {
	return m.register
}

func (m *Module) ValidateUsecase() *Validate {
	return m.validate
}
