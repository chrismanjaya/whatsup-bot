package usecase

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/message"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

// summaryKeywords are a cheap pre-filter: only messages containing one of them
// are sent to the model to be checked for a summary request.
var summaryKeywords = []string{"summary", "summarize", "summarise", "report", "rangkum", "ringkas", "laporan", "rekap"}

type SummarizeTransactionsUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	parser   port.MessageParser
}

func NewSummarizeTransactionsUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, parser port.MessageParser) *SummarizeTransactionsUseCase {
	return &SummarizeTransactionsUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser}
}

// Execute handles a request such as "this month report". handled is false when
// the message isn't a summary request, so the caller can fall through to the
// other flows.
func (uc *SummarizeTransactionsUseCase) Execute(ctx context.Context, senderJID, rawText string) (reply string, handled bool, err error) {
	if !containsAny(rawText, summaryKeywords) {
		return "", false, nil
	}

	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return "", false, utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil {
		return "", false, utils.Wrap(constant.ErrNotFound, "user not registered")
	}

	intent, err := uc.parser.ParseSummary(ctx, rawText)
	if err != nil {
		return "", false, utils.Wrap(err, "parse summary failed")
	}
	if !intent.IsSummary {
		return "", false, nil
	}

	loc := jakartaLocation()
	start, startErr := time.ParseInLocation("2006-01-02", intent.StartDate, loc)
	end, endErr := time.ParseInLocation("2006-01-02", intent.EndDate, loc)
	if startErr != nil || endErr != nil {
		return message.SummaryDateUnclear, true, nil
	}
	if end.Before(start) {
		start, end = end, start
	}
	// The range is inclusive, so 1 Aug - 31 Aug is one month but 1 Aug - 1 Sep is not.
	if !end.Before(start.AddDate(0, 1, 0)) {
		return message.SummaryRangeTooLong, true, nil
	}

	// A negative limit means no limit: a month of transactions fits in one message.
	txs, _, err := uc.txRepo.ListByUserBetween(ctx, user.ID, start, end.AddDate(0, 0, 1), -1, 0)
	if err != nil {
		return "", true, utils.WrapStd(constant.ErrInternal, "list transactions failed", err)
	}

	return renderSummary(start, end, txs), true, nil
}

// side aggregates one direction (expense or income) of a period.
type side struct {
	total    int64
	count    int
	biggest  *domain.Transaction
	byCat    map[domain.Category]int
	catTotal map[domain.Category]int64
}

func newSide() *side {
	return &side{byCat: map[domain.Category]int{}, catTotal: map[domain.Category]int64{}}
}

func (s *side) add(tx *domain.Transaction) {
	s.total += tx.Amount
	s.count++
	if s.biggest == nil || tx.Amount > s.biggest.Amount {
		s.biggest = tx
	}
	s.byCat[tx.Category]++
	s.catTotal[tx.Category] += tx.Amount
}

// topCategory is the category with the largest total amount; ties go to the
// higher count and then the name, so the result is deterministic.
func (s *side) topCategory() domain.Category {
	cats := make([]domain.Category, 0, len(s.byCat))
	for c := range s.byCat {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool {
		a, b := cats[i], cats[j]
		if s.catTotal[a] != s.catTotal[b] {
			return s.catTotal[a] > s.catTotal[b]
		}
		if s.byCat[a] != s.byCat[b] {
			return s.byCat[a] > s.byCat[b]
		}
		return a < b
	})
	return cats[0]
}

// renderSummary builds the summary of txs (oldest first) for the inclusive
// range [start, end]. Transactions still waiting for an amount are skipped.
func renderSummary(start, end time.Time, txs []*domain.Transaction) string {
	expense, income := newSide(), newSide()
	counted := make([]*domain.Transaction, 0, len(txs))
	for _, tx := range txs {
		if tx.Amount <= 0 {
			continue
		}
		counted = append(counted, tx)
		if tx.Type == domain.Income {
			income.add(tx)
		} else {
			expense.add(tx)
		}
	}

	period := "[" + start.Format("2/1/06") + " - " + end.Format("2/1/06") + "]"
	if len(counted) == 0 {
		return fmt.Sprintf(message.QueryNoResults, period)
	}

	var insights []string
	if expense.count > 0 {
		insights = append(insights, fmt.Sprintf(message.SummaryTopExpense, titleCase(expense.topCategory().String())))
	}
	if income.count > 0 {
		insights = append(insights, fmt.Sprintf(message.SummaryTopIncome, titleCase(income.topCategory().String())))
	}
	if expense.count > 0 {
		b := expense.biggest
		insights = append(insights, fmt.Sprintf(message.SummaryBigExpense, titleCase(b.Description), message.Currency, formatAmount(b.Amount)))
	}
	if income.count > 0 {
		b := income.biggest
		insights = append(insights, fmt.Sprintf(message.SummaryBigIncome, titleCase(b.Description), message.Currency, formatAmount(b.Amount)))
	}

	lines := make([]string, 0, len(counted))
	for _, tx := range counted {
		sign := "-"
		if tx.Type == domain.Income {
			sign = "+"
		}
		lines = append(lines, fmt.Sprintf(message.SummaryDetailLine,
			tx.TransactionDate.Format("2/1"), sign, formatAmount(tx.Amount), titleCase(tx.Description)))
	}

	return strings.NewReplacer(
		"[[period]]", period,
		"[[expense]]", renderSide(expense),
		"[[income]]", renderSide(income),
		"[[insights]]", strings.Join(insights, ", and "),
		"[[count]]", strconv.Itoa(len(counted)),
		"[[transactions_word]]", pluralTransactions(len(counted)),
		"[[details]]", strings.Join(lines, "\n"),
	).Replace(message.SummaryTemplate)
}

func renderSide(s *side) string {
	if s.count == 0 {
		return message.SummaryNone
	}
	return fmt.Sprintf(message.SummaryTotal, message.Currency, formatAmount(s.total), s.count, pluralTransactions(s.count))
}

func pluralTransactions(n int) string {
	if n == 1 {
		return "transaction"
	}
	return "transactions"
}
