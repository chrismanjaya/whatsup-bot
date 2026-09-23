package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type ComputeSplitUseCase struct {
	groupRepo port.GroupRepository
	txRepo    port.TransactionRepository
}

func NewComputeSplitUseCase(groupRepo port.GroupRepository, txRepo port.TransactionRepository) *ComputeSplitUseCase {
	return &ComputeSplitUseCase{groupRepo: groupRepo, txRepo: txRepo}
}

func (uc *ComputeSplitUseCase) Execute(ctx context.Context, groupID int64) (string, error) {
	members, err := uc.groupRepo.ListMembers(ctx, groupID)
	if err != nil {
		return "", utils.WrapStd(constant.ErrInternal, "list members failed", err)
	}
	if len(members) == 0 {
		return "No members found for this group.", nil
	}

	startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

	txs, err := uc.txRepo.FindByGroup(ctx, groupID, startOfMonth)
	if err != nil {
		return "", utils.WrapStd(constant.ErrInternal, "fetch transactions failed", err)
	}

	totals := make(map[int64]int64)
	for _, m := range members {
		totals[m.ID] = 0
	}

	var totalAll int64
	for _, tx := range txs {
		if tx.Type == domain.Outcome {
			totals[tx.UserID] += tx.Amount
			totalAll += tx.Amount
		}
	}

	share := totalAll / int64(len(members))

	var sb strings.Builder
	sb.WriteString("This month's household expenses:\n")
	for _, m := range members {
		sb.WriteString(fmt.Sprintf("%s: Rp%d\n", m.Name, totals[m.ID]))
	}
	sb.WriteString(fmt.Sprintf("Total: Rp%d (Rp%d each for equal split)\n", totalAll, share))

	for _, m := range members {
		if diff := totals[m.ID] - share; diff < 0 {
			sb.WriteString(fmt.Sprintf("%s owes Rp%d to balance out\n", m.Name, -diff))
		}
	}

	return sb.String(), nil
}
