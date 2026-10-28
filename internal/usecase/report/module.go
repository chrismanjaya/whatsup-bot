package report

type Module struct {
	getDetail          *GetDetail
	getSummary         *GetSummary
	getTransactionList *GetTransactionList
}

func NewModule(
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	geminiClient GeminiClient,
) *Module {
	return &Module{
		getDetail:          NewGetDetail(transactionRepo, userRepo, geminiClient),
		getSummary:         NewGetSummary(transactionRepo, userRepo, geminiClient),
		getTransactionList: NewGetTransactionList(transactionRepo, userRepo),
	}
}

func (m *Module) GetDetailUsecase() *GetDetail {
	return m.getDetail
}

func (m *Module) GetSummarylUsecase() *GetSummary {
	return m.getSummary
}

func (m *Module) GetTransactionListUsecase() *GetTransactionList {
	return m.getTransactionList
}
