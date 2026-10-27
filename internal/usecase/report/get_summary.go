package report

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type GetSummary struct {
	transactionRepo TransactionRepository
	userRepo        UserRepository

	geminiClient GeminiClient
}

func NewGetSummary(
	transactionRepo TransactionRepository,
	userRepo UserRepository,

	geminiClient GeminiClient,
) *GetSummary {
	return &GetSummary{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		geminiClient:    geminiClient,
	}
}

type (
	SummaryStatus int32

	SummaryResult struct {
		TransactionPeriod domain.TransactionPeriod
		ExpenseInfo       *SummaryInfo
		IncomeInfo        *SummaryInfo
		Transactions      []*domain.Transaction
	}

	SummaryInfo struct {
		Total              int64
		Count              int
		HighestTransaction domain.Transaction
		TotalPerCategories map[domain.Category]int64
	}
)

const (
	defaultGetSummaryLimit  = -1
	defaultGetSummaryOffset = 0
)

func NewSummaryInfo() *SummaryInfo {
	return &SummaryInfo{
		TotalPerCategories: make(map[domain.Category]int64),
	}
}

func (s *SummaryInfo) Add(transaction *domain.Transaction) {
	s.Total += transaction.Amount
	s.Count += 1

	if s.HighestTransaction.Amount < transaction.Amount {
		s.HighestTransaction = *transaction
	}

	s.TotalPerCategories[transaction.Category] += transaction.Amount
}

func (s *SummaryInfo) GetHighestTransactionAmount() int64 {
	return s.HighestTransaction.Amount
}

func (s *SummaryInfo) GetTopTransaction() (domain.Category, int64) {
	var (
		topCategory      domain.Category
		totalTopCategory int64
	)

	for category, total := range s.TotalPerCategories {
		if totalTopCategory < total {
			topCategory = category
			totalTopCategory = total
		}
	}

	return topCategory, totalTopCategory
}

func (uc *GetSummary) Execute(ctx context.Context, rawText string, user *domain.User) (*SummaryResult, error) {
	// validate user
	isValidUser, userID := ValidateUser(ctx, user)
	if !isValidUser {
		return nil, util.Wrap(constant.ErrInvalidUser, "invalid user")
	}

	// parse message
	message, err := uc.geminiClient.ParseReportSummaryMessage(ctx, rawText)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "parse message failed", err)
	}

	if !message.Valid || message.ReportMessage == nil {
		return nil, util.Wrap(constant.ErrInvalidRequest, "message is not a contain report information")
	}

	// generate transaction period
	transactionPeriod := generateTransactionPeriod(message.ReportMessage)
	if transactionPeriod == nil {
		return nil, util.Wrap(constant.ErrUnclearReportPeriod, "invalid transaction period")
	}
	if !ValidateTransactionRange(transactionPeriod) {
		return nil, util.Wrap(constant.ErrInvalidReportPeriodRange, "invalid transaction range")
	}

	// get transaction
	trarnsactions, _, err := uc.transactionRepo.GetListByPeriod(
		ctx,
		userID,
		transactionPeriod.Start,
		transactionPeriod.End,
		defaultGetSummaryLimit,
		defaultGetSummaryOffset,
	)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "list transactions failed", err)
	}

	if len(trarnsactions) == 0 {
		return nil, util.Wrap(constant.ErrReportNotFound, "transaction not found")
	}

	// generate summarize
	return sumarize(trarnsactions, *transactionPeriod), nil
}

func sumarize(transactions []*domain.Transaction, transactionPeriod domain.TransactionPeriod) *SummaryResult {
	var (
		calculatedTransaction = make([]*domain.Transaction, 0, len(transactions))
		expenseInfo           = NewSummaryInfo()
		incomeInfo            = NewSummaryInfo()
	)

	for _, transaction := range transactions {
		if transaction == nil {
			continue
		}

		calculatedTransaction = append(calculatedTransaction, transaction)

		switch transaction.Type {
		case domain.Expense:
			expenseInfo.Add(transaction)
		case domain.Income:
			incomeInfo.Add(transaction)
		}
	}

	return &SummaryResult{
		TransactionPeriod: transactionPeriod,
		ExpenseInfo:       expenseInfo,
		IncomeInfo:        incomeInfo,
		Transactions:      calculatedTransaction,
	}
}
