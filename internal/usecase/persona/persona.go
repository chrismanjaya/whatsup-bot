package persona

type Module struct {
	picker *Picker
}

func NewModule() *Module {
	return &Module{
		picker: NewPicker(),
	}
}

func (m *Module) GetPickerUsecase() *Picker {
	return m.picker
}
