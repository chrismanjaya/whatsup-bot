package usecase

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/message"
	"whatsup-bot/internal/persona"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type RecordTransactionUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	parser   port.MessageParser
	persona  *persona.Picker
}

func NewRecordTransactionUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, parser port.MessageParser, picker *persona.Picker) *RecordTransactionUseCase {
	return &RecordTransactionUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser, persona: picker}
}

// Execute records a transaction from rawText and returns the replies to
// send: the confirmation (linked to the transaction, so replying to it can
// update or delete it), sometimes followed by Frankie's quip as a separate
// message.
func (uc *RecordTransactionUseCase) Execute(ctx context.Context, senderJID, rawText string, isShared bool, groupID int64) ([]Reply, error) {
	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return nil, utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil {
		return nil, utils.Wrap(constant.ErrNotFound, "user not registered")
	}

	parsed, err := uc.parser.Parse(ctx, rawText)
	if err != nil {
		return nil, utils.Wrap(err, "parse failed")
	}
	if !parsed.Valid {
		return nil, utils.Wrap(constant.ErrInvalidRequest, "message is not a financial transaction")
	}

	txType, err := domain.ParseTransactionType(parsed.Type)
	if err != nil {
		return nil, utils.WrapStd(constant.ErrInternal, "invalid type from parser", err)
	}

	description := parsed.Description
	if description == "" {
		description = rawText
	}

	category, catErr := domain.ParseCategory(parsed.Category)
	if catErr != nil {
		slog.Warn("invalid category from gemini, defaulting to other", "category", parsed.Category, "error", catErr)
		category = domain.CategoryOther
	}

	tx := &domain.Transaction{
		UserID:          user.ID,
		Description:     description,
		Type:            txType,
		Amount:          parsed.Amount,
		Category:        category,
		IsShared:        isShared,
		GroupID:         groupID,
		TransactionDate: resolveTransactionDate(parsed.Date, time.Now()),
		CreatedAt:       time.Now(),
	}

	greeting := uc.dailyGreeting(ctx, user, senderJID)

	if err := uc.txRepo.Save(ctx, tx); err != nil {
		return nil, utils.WrapStd(constant.ErrInternal, "save failed", err)
	}

	var replies []Reply
	if greeting != "" {
		replies = append(replies, Reply{Text: greeting})
	}
	if tx.Amount <= 0 {
		return append(replies, Reply{Text: renderAmountPrompt(tx), Tx: tx}), nil
	}
	return append(replies, transactionReplies(tx, newTransactionQuip(uc.persona, parsed.Quip, tx, senderJID))...), nil
}

// dailyGreeting returns Frankie's greeting if this is the user's first
// transaction today (Asia/Jakarta), or "". It's based on the stored
// transactions, so it survives restarts and deploys. A lookup error just
// skips the greeting.
func (uc *RecordTransactionUseCase) dailyGreeting(ctx context.Context, user *domain.User, senderJID string) string {
	if uc.persona == nil {
		return ""
	}
	last, ok, err := uc.txRepo.LastCreatedAt(ctx, user.ID)
	if err != nil {
		utils.LogWarn("last transaction lookup failed, skipping greeting", err, "user_id", user.ID)
		return ""
	}
	if ok && persona.SameDay(last, uc.persona.Now()) {
		return ""
	}
	return uc.persona.DailyGreeting(senderJID, titleCase(user.Name))
}

// transactionReplies is a transaction confirmation (linked to tx) followed
// by quip as its own message (starting with message.QuipPrefix), if there
// is one. The quip is deliberately not linked to tx: a transaction holds one
// wa_message_id, which must stay the confirmation's so replying to it keeps
// working. A reply to the quip is recognized by its prefix in the router.
func transactionReplies(tx *domain.Transaction, quip string) []Reply {
	replies := []Reply{{Text: renderTransactionReply(tx), Tx: tx}}
	if quip != "" {
		replies = append(replies, Reply{Text: message.QuipPrefix + quip, Typing: true})
	}
	return replies
}

// newTransactionQuip returns Frankie's comment for a newly completed
// transaction, or "" when it rolls no quip: income always gets one,
// expenses message.QuipExpenseChancePct% of the time. It prefers the
// model's quip (modelQuip, may be "") and falls back to the pooled ones.
func newTransactionQuip(p *persona.Picker, modelQuip string, tx *domain.Transaction, senderJID string) string {
	if p == nil {
		return ""
	}
	isIncome := tx.Type == domain.Income
	chance := message.QuipExpenseChancePct
	if isIncome {
		chance = message.QuipIncomeChancePct
	}
	if !p.Chance(chance) {
		return ""
	}
	if q := persona.SanitizeQuip(modelQuip); q != "" {
		// The prompt asks for English, but the model sometimes follows the
		// user's Indonesian anyway; fall back to an English pool quip then.
		if persona.IsEnglish(q) {
			return q
		}
		slog.Info("dropped non-English quip from model", "quip", q)
	}
	return p.FallbackQuip(tx.Category.String(), isIncome, senderJID)
}

// resolveTransactionDate parses a "YYYY-MM-DD" date string from the model
// into Asia/Jakarta local time, falling back to fallback when the string is
// empty or fails to parse.
func resolveTransactionDate(dateStr string, fallback time.Time) time.Time {
	if dateStr == "" {
		return fallback
	}
	loc, locErr := time.LoadLocation("Asia/Jakarta")
	if locErr != nil {
		loc = time.UTC
	}
	d, dateErr := time.ParseInLocation("2006-01-02", dateStr, loc)
	if dateErr != nil {
		utils.LogWarn("failed to parse date from gemini, using fallback", dateErr, "date", dateStr)
		return fallback
	}
	return d
}

func renderDeletedReply(tx *domain.Transaction) string {
	replacer := strings.NewReplacer(
		"[[transaction_description]]", titleCase(tx.Description),
		"[[transaction_amount_formatted]]", message.Currency+" "+formatAmount(tx.Amount),
	)
	return replacer.Replace(message.DeletedReplyTemplate)
}

// renderAmountPrompt asks for the missing amount of a transaction saved with
// amount 0. Replying to it goes through the amend flow, which fills it in.
func renderAmountPrompt(tx *domain.Transaction) string {
	replacer := strings.NewReplacer(
		"[[transaction_description]]", titleCase(tx.Description),
		"[[transaction_category]]", titleCase(tx.Category.String()),
		"[[transaction_date_formatted]]", tx.TransactionDate.Format("02 Jan 2006"),
	)
	return replacer.Replace(message.AmountPromptTemplate)
}

func renderTransactionReply(tx *domain.Transaction) string {
	typeStr := message.TypeExpense
	if tx.Type == domain.Income {
		typeStr = message.TypeIncome
	}

	replacer := strings.NewReplacer(
		"[[transaction_type_str]]", typeStr,
		"[[transaction_description]]", titleCase(tx.Description),
		"[[transaction_category]]", titleCase(tx.Category.String()),
		"[[transaction_amount_formatted]]", message.Currency+" "+formatAmount(tx.Amount),
		"[[transaction_date_formatted]]", tx.TransactionDate.Format("02 Jan 2006"),
	)
	return replacer.Replace(message.TransactionReplyTemplate)
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func formatAmount(amount int64) string {
	s := strconv.FormatInt(amount, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}

	var sb strings.Builder
	first := n % 3
	if first == 0 {
		first = 3
	}
	sb.WriteString(s[:first])
	for i := first; i < n; i += 3 {
		sb.WriteString(",")
		sb.WriteString(s[i : i+3])
	}

	if neg {
		return "-" + sb.String()
	}
	return sb.String()
}
