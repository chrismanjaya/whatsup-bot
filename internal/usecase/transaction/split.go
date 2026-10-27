package transaction

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Split struct {
	groupRepo       GroupRepository
	transactionRepo TransactionRepository
	userRepo        UserRepository

	geminiClient GeminiClient
}

func NewSplit(
	groupRepo GroupRepository,
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	geminiClient GeminiClient,
) *Split {
	return &Split{
		groupRepo:       groupRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		geminiClient:    geminiClient,
	}
}

type (
	SplitResult struct {
		MemberShares []MemberShare
		Total        int64 // total amount
		Share        int64 // total amount per member
	}

	MemberShare struct {
		Name  string
		Spent int64
		Owes  int64
	}
)

func (uc *Split) Execute(ctx context.Context, group *domain.Group) (*SplitResult, error) {
	// validate group
	isValidGroup, groupID := ValidateGroup(group)
	if !isValidGroup {
		return nil, util.Wrap(constant.ErrInvalidGroup, "invalid group")
	}

	// get members
	members, err := uc.groupRepo.GetListMember(ctx, groupID)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "get list members failed", err)
	}

	if len(members) == 0 {
		return nil, util.Wrap(constant.ErrInvalidGroup, "invalid members")
	}

	// find transactions
	groupTransactions, err := uc.transactionRepo.FindByGroup(ctx, groupID, util.GetStartOfMonth())
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "fetch transactions failed", err)
	}

	if len(groupTransactions) == 0 {
		return nil, util.Wrap(constant.ErrTransactionNotFound, "transaction not found")
	}

	return doSplitTransaction(groupTransactions, members)
}

func doSplitTransaction(groupTransactions []*domain.Transaction, members []*domain.User) (*SplitResult, error) {
	var (
		totalExpenses int64 = 0
		totalMember   int   = len(members)
		totalShares   int64 = 0

		memberExpenses = make(map[int64]int64, totalMember)
	)

	for _, transaction := range groupTransactions {
		if transaction.Type != domain.Expense {
			continue
		}

		memberExpenses[transaction.UserID] += transaction.Amount
		totalExpenses += transaction.Amount
	}

	totalShares = totalExpenses / int64(totalMember)

	var memberShares []MemberShare
	for _, member := range members {
		memberShares = append(memberShares, MemberShare{
			Name:  member.Name,
			Spent: memberExpenses[member.ID],
			Owes:  memberExpenses[member.ID] - totalShares,
		})
	}

	return &SplitResult{
		MemberShares: memberShares,
		Total:        totalExpenses,
		Share:        totalShares,
	}, nil
}
