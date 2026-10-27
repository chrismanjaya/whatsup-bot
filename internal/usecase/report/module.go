package report

type Module struct {
	getDetail  *GetDetail
	getSummary *GetSummary
}

func NewModule(
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	geminiClient GeminiClient,
) *Module {
	return &Module{
		getDetail:  NewGetDetail(transactionRepo, userRepo, geminiClient),
		getSummary: NewGetSummary(transactionRepo, userRepo, geminiClient),
	}
}

func (m *Module) GetDetailUsecase() *GetDetail {
	return m.getDetail
}

func (m *Module) GetSummarylUsecase() *GetSummary {
	return m.getSummary
}
