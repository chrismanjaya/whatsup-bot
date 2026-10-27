package transaction

type Module struct {
	record *Record
	amend  *Amend
	split  *Split
}

func NewModule(
	groupRepo GroupRepository,
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	geminiClient GeminiClient,
) Module {
	return Module{
		record: NewRecord(groupRepo, transactionRepo, userRepo, geminiClient),
		amend:  NewAmend(groupRepo, transactionRepo, userRepo, geminiClient),
		split:  NewSplit(groupRepo, transactionRepo, userRepo, geminiClient),
	}
}

func (m *Module) RecordUsecase() *Record {
	return m.record
}

func (m *Module) AmendUsecase() *Amend {
	return m.amend
}

func (m *Module) SplitUsecase() *Split {
	return m.split
}
