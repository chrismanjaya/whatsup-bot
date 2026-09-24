package usecase

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/message"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

// maxPageNumber bounds a user-typed page number so the paging offset can't
// overflow; anything above it is reported as out of range.
const maxPageNumber = 1_000_000

var pageNumberRe = regexp.MustCompile(`^(?:(?:page|halaman|hal|hlm|p)\s*-?\s*)?(\d+)$`)

type PageTransactionsUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	pageRepo port.QueryPageRepository
}

func NewPageTransactionsUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, pageRepo port.QueryPageRepository) *PageTransactionsUseCase {
	return &PageTransactionsUseCase{userRepo: userRepo, txRepo: txRepo, pageRepo: pageRepo}
}

// Execute handles a reply to a page summary message identified by
// waMessageID ("next", "prev", "page 3", ...). handled is false when
// waMessageID isn't a page summary the bot sent, so the caller can fall back
// to normal handling.
func (uc *PageTransactionsUseCase) Execute(ctx context.Context, senderJID, waMessageID, rawText string) (replies []Reply, handled bool, err error) {
	qp, err := uc.pageRepo.FindByWAMessageID(ctx, waMessageID)
	if err != nil {
		return nil, false, utils.WrapStd(constant.ErrInternal, "lookup query page failed", err)
	}
	if qp == nil {
		return nil, false, nil
	}

	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return nil, true, utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil || user.ID != qp.UserID {
		return []Reply{{Text: message.PageNotOwner}}, true, nil
	}

	target, ok := parsePageCommand(rawText, qp.Page)
	if !ok {
		return []Reply{{Text: message.PageUnclear}}, true, nil
	}
	if target < 1 || target > maxPageNumber {
		// Report against the real page count, taken from the current page's query.
		_, total, err := uc.txRepo.ListByUserBetween(ctx, qp.UserID, qp.From, qp.To, 1, 0)
		if err != nil {
			return nil, true, utils.WrapStd(constant.ErrInternal, "count transactions failed", err)
		}
		return []Reply{{Text: pageOutOfRange(target, (total+queryPageSize-1)/queryPageSize)}}, true, nil
	}

	replies, err = listPage(ctx, uc.txRepo, uc.pageRepo, qp.UserID, qp.From, qp.To, target)
	if err != nil {
		return nil, true, err
	}
	return replies, true, nil
}

// parsePageCommand interprets a paging reply relative to the current page.
// It reports false when the text isn't a recognised paging command.
func parsePageCommand(text string, current int) (int, bool) {
	t := strings.ToLower(strings.TrimSpace(text))
	t = strings.TrimRight(t, ".!")

	switch t {
	case "next", "n", "lanjut", "lanjutkan", "selanjutnya", "berikutnya":
		return current + 1, true
	case "prev", "previous", "back", "sebelumnya", "sebelum", "kembali":
		return current - 1, true
	}
	if m := pageNumberRe.FindStringSubmatch(t); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}
