package report

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type GetTransactionList struct {
	transactionRepo TransactionRepository
	userRepo        UserRepository
}

func NewGetTransactionList(
	transactionRepo TransactionRepository,
	userRepo UserRepository,
) *GetTransactionList {
	return &GetTransactionList{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
	}
}

type (
	TransactionListRequest struct {
		UserID int64
		Period domain.TransactionPeriod
		Type   domain.TransactionType // empty means all transaction types
		Limit  int
		Offset int
	}

	TransactionListResult struct {
		User         *domain.User
		Period       domain.TransactionPeriod
		Transactions []*domain.Transaction
		TotalCount   int
		Limit        int
		Offset       int
	}
)

const (
	defaultGetTransactionListLimit = 50
	maxGetTransactionListLimit     = 200

	maxTransactionListRangeMonth = 6
)

func (uc *GetTransactionList) Execute(ctx context.Context, req TransactionListRequest) (*TransactionListResult, error) {
	// validate request
	if req.UserID <= 0 {
		return nil, util.Wrap(constant.ErrInvalidUser, "invalid user")
	}

	if req.Type != "" && !req.Type.Valid() {
		return nil, util.Wrap(constant.ErrInvalidTransactionType, "invalid transaction type")
	}

	if req.Period.Start.IsZero() || req.Period.End.IsZero() || req.Period.End.Before(req.Period.Start) {
		return nil, util.Wrap(constant.ErrUnclearReportPeriod, "invalid transaction period")
	}

	// period must not exceed 6 month
	if req.Period.End.After(req.Period.Start.AddDate(0, maxTransactionListRangeMonth, 0)) {
		return nil, util.Wrap(constant.ErrInvalidReportPeriodRange, "transaction period exceed 6 month")
	}

	if req.Offset < 0 {
		return nil, util.Wrap(constant.ErrInvalidRequest, "negative offset")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultGetTransactionListLimit
	}
	limit = min(limit, maxGetTransactionListLimit)

	// get user
	user, err := uc.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "find user failed", err)
	}

	if user == nil {
		return nil, util.Wrap(constant.ErrUserNotRegister, "user not found")
	}

	// get list of transactions
	transactions, totalCount, err := uc.transactionRepo.GetListByPeriod(
		ctx,
		user.ID,
		req.Period.Start,
		req.Period.End,
		req.Type,
		limit,
		req.Offset,
	)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "list transactions failed", err)
	}

	return &TransactionListResult{
		User:         user,
		Period:       req.Period,
		Transactions: transactions,
		TotalCount:   totalCount,
		Limit:        limit,
		Offset:       req.Offset,
	}, nil
}
