package transaction

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/parser"
	"whatsup-bot/internal/repository"
	"whatsup-bot/internal/usecase/message"
	"whatsup-bot/internal/usecase/response"
	"whatsup-bot/internal/util"
)

// queryPageSize is how many transactions go out per page, one WhatsApp
// message each so every one can be replied to for update or delete.
const queryPageSize = 10

// queryKeywords are a cheap pre-filter: only messages containing one of them
// are sent to the model to be checked for a listing request, so ordinary
// transaction messages don't cost an extra model call.
var queryKeywords = []string{"transaksi", "transaction", "riwayat", "history", "daftar", "list"}

type QueryUseCase struct {
	userRepo repository.UserRepository
	txRepo   repository.TransactionRepository
	pageRepo repository.QueryPageRepository
	parser   parser.MessageParser
}

func NewQueryUseCase(userRepo repository.UserRepository, txRepo repository.TransactionRepository, pageRepo repository.QueryPageRepository, parser parser.MessageParser) *QueryUseCase {
	return &QueryUseCase{userRepo: userRepo, txRepo: txRepo, pageRepo: pageRepo, parser: parser}
}

// Execute handles a request such as "transaksi tanggal 1 agustus". handled
// is false when the message isn't a listing request, so the caller can fall
// back to treating it as a new transaction.
func (uc *QueryUseCase) Execute(ctx context.Context, senderJID, rawText string) (replies []response.Reply, handled bool, err error) {
	if !looksLikeQuery(rawText) {
		return nil, false, nil
	}

	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return nil, false, util.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil {
		return nil, false, util.Wrap(constant.ErrNotFound, "user not registered")
	}

	intent, err := uc.parser.ParseQuery(ctx, rawText)
	if err != nil {
		return nil, false, util.Wrap(err, "parse query failed")
	}
	if !intent.IsQuery {
		return nil, false, nil
	}

	loc := jakartaLocation()
	start, startErr := time.ParseInLocation("2006-01-02", intent.StartDate, loc)
	end, endErr := time.ParseInLocation("2006-01-02", intent.EndDate, loc)
	if startErr != nil || endErr != nil {
		return []response.Reply{{Text: message.QueryDateUnclear}}, true, nil
	}
	if end.Before(start) {
		start, end = end, start
	}

	replies, err = uc.listPage(ctx, user.ID, start, end.AddDate(0, 0, 1), 1)
	if err != nil {
		return nil, true, err
	}
	return replies, true, nil
}

func (uc *QueryUseCase) listPage(ctx context.Context, userID int64, from, to time.Time, page int) ([]response.Reply, error) {
	return listPage(ctx, uc.txRepo, uc.pageRepo, userID, from, to, page)
}

// listPage builds the messages for one page of the user's transactions in
// [from, to): one message per transaction, then a summary with paging
// instructions when there is more than one page.
func listPage(ctx context.Context, txRepo repository.TransactionRepository, pageRepo repository.QueryPageRepository, userID int64, from, to time.Time, page int) ([]response.Reply, error) {
	period := formatPeriod(from, to)

	txs, total, err := txRepo.ListByUserBetween(ctx, userID, from, to, queryPageSize, (page-1)*queryPageSize)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "list transactions failed", err)
	}
	if total == 0 {
		return []response.Reply{{Text: fmt.Sprintf(message.QueryNoResults, period)}}, nil
	}

	totalPages := (total + queryPageSize - 1) / queryPageSize
	if page > totalPages {
		return []response.Reply{{Text: pageOutOfRange(page, totalPages)}}, nil
	}

	replies := make([]response.Reply, 0, len(txs)+1)
	for _, tx := range txs {
		replies = append(replies, response.Reply{Text: renderTransactionReply(tx), Tx: tx})
	}
	if totalPages == 1 {
		return replies, nil
	}

	qp := &domain.QueryPage{UserID: userID, From: from, To: to, Page: page, CreatedAt: time.Now()}
	if err := pageRepo.Save(ctx, qp); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "save query page failed", err)
	}

	shown := (page-1)*queryPageSize + len(txs)
	summary := strings.NewReplacer(
		"[[page]]", strconv.Itoa(page),
		"[[total_pages]]", strconv.Itoa(totalPages),
		"[[period]]", period,
		"[[from_n]]", strconv.Itoa((page-1)*queryPageSize+1),
		"[[to_n]]", strconv.Itoa(shown),
		"[[total]]", strconv.Itoa(total),
		"[[remaining]]", strconv.Itoa(total-shown),
	).Replace(message.PageSummaryTemplate)
	return append(replies, response.Reply{Text: summary, Page: qp}), nil
}

func looksLikeQuery(text string) bool {
	return containsAny(text, queryKeywords)
}

// containsAny reports whether text contains any of keywords, ignoring case.
func containsAny(text string, keywords []string) bool {
	lower := strings.ToLower(text)
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func pageOutOfRange(requested, totalPages int) string {
	return fmt.Sprintf(message.PageOutOfRange, requested, totalPages)
}

// formatPeriod renders [from, to) as a single date, or "from – to" (inclusive).
func formatPeriod(from, to time.Time) string {
	const layout = "02 Jan 2006"
	if from.Year() <= 1970 {
		return "all time"
	}
	last := to.AddDate(0, 0, -1)
	if !last.After(from) {
		return from.Format(layout)
	}
	return from.Format(layout) + " – " + last.Format(layout)
}

func jakartaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.UTC
	}
	return loc
}
