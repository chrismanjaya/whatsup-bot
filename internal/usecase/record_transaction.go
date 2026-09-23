package usecase

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type RecordTransactionUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	parser   port.MessageParser
}

func NewRecordTransactionUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, parser port.MessageParser) *RecordTransactionUseCase {
	return &RecordTransactionUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser}
}

func (uc *RecordTransactionUseCase) Execute(ctx context.Context, senderJID, rawText string, isShared bool, groupID int64) (reply string, tx *domain.Transaction, err error) {
	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return "", nil, utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil {
		return "", nil, utils.Wrap(constant.ErrNotFound, "user not registered")
	}

	parsed, err := uc.parser.Parse(ctx, rawText)
	if err != nil {
		return "", nil, utils.Wrap(err, "parse failed")
	}
	if !parsed.Valid {
		return "", nil, utils.Wrap(constant.ErrInvalidRequest, "message is not a financial transaction")
	}

	txType, err := domain.ParseTransactionType(parsed.Type)
	if err != nil {
		return "", nil, utils.WrapStd(constant.ErrInternal, "invalid type from parser", err)
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

	tx = &domain.Transaction{
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

	if err := uc.txRepo.Save(ctx, tx); err != nil {
		return "", nil, utils.WrapStd(constant.ErrInternal, "save failed", err)
	}

	return renderTransactionReply(tx), tx, nil
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

const transactionReplyTemplate = `*[[transaction_type_str]]*
- Desc: *[[transaction_description]]*
- Category: *[[transaction_category]]*
- Amount: *[[transaction_amount_formatted]]*
- Date: *[[transaction_date_formatted]]*

_*Reply to this message to update or delete this transaction_`

const deletedReplyTemplate = `*DELETED*
- Desc: *[[transaction_description]]*
- Amount: *[[transaction_amount_formatted]]*`

func renderDeletedReply(tx *domain.Transaction) string {
	replacer := strings.NewReplacer(
		"[[transaction_description]]", titleCase(tx.Description),
		"[[transaction_amount_formatted]]", "IDR "+formatAmount(tx.Amount),
	)
	return replacer.Replace(deletedReplyTemplate)
}

func renderTransactionReply(tx *domain.Transaction) string {
	typeStr := "EXPENSE"
	if tx.Type == domain.Income {
		typeStr = "INCOME"
	}

	replacer := strings.NewReplacer(
		"[[transaction_type_str]]", typeStr,
		"[[transaction_description]]", titleCase(tx.Description),
		"[[transaction_category]]", titleCase(tx.Category.String()),
		"[[transaction_amount_formatted]]", "IDR "+formatAmount(tx.Amount),
		"[[transaction_date_formatted]]", tx.TransactionDate.Format("02 Jan 2006"),
	)
	return replacer.Replace(transactionReplyTemplate)
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
