package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/port"
)

type RecordTransactionUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	parser   port.MessageParser
}

func NewRecordTransactionUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, parser port.MessageParser) *RecordTransactionUseCase {
	return &RecordTransactionUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser}
}

func (uc *RecordTransactionUseCase) Execute(ctx context.Context, senderJID, rawText string, isShared bool, groupID int64) (reply string, recorded bool, err error) {
	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return "", false, err
	}
	if user == nil {
		return "Please register first: register <name> <email>", false, nil
	}

	parsed, err := uc.parser.Parse(ctx, rawText)
	if err != nil {
		return "", false, fmt.Errorf("parse failed: %w", err)
	}
	if !parsed.Valid {
		return "I can only help track income and expenses. Try: \"spent 50k on lunch\"", false, nil
	}

	txType, err := domain.ParseTransactionType(parsed.Type)
	if err != nil {
		return "", false, fmt.Errorf("invalid type from parser: %w", err)
	}

	description := parsed.Description
	if description == "" {
		description = rawText
	}

	transactionDate := time.Now()
	if parsed.Date != "" {
		loc, locErr := time.LoadLocation("Asia/Jakarta")
		if locErr != nil {
			loc = time.UTC
		}
		if d, dateErr := time.ParseInLocation("2006-01-02", parsed.Date, loc); dateErr == nil {
			transactionDate = d
		} else {
			slog.Warn("failed to parse date from gemini, using now", "date", parsed.Date, "error", dateErr)
		}
	}

	tx := &domain.Transaction{
		UserID:          user.ID,
		Description:     description,
		Type:            txType,
		Amount:          parsed.Amount,
		Category:        parsed.Category,
		IsShared:        isShared,
		GroupID:         groupID,
		TransactionDate: transactionDate,
		CreatedAt:       time.Now(),
	}

	if err := uc.txRepo.Save(ctx, tx); err != nil {
		return "", false, fmt.Errorf("save failed: %w", err)
	}

	return renderTransactionReply(tx), true, nil
}

const transactionReplyTemplate = `*[[transaction_type_str]]*
- Desc: *[[transaction_description]]*
- Category: *[[transaction_category]]*
- Amount: *[[transaction_amount_formatted]]*
- Date: *[[transaction_date_formatted]]*`

func renderTransactionReply(tx *domain.Transaction) string {
	typeStr := "EXPENSE"
	if tx.Type == domain.Income {
		typeStr = "INCOME"
	}

	replacer := strings.NewReplacer(
		"[[transaction_type_str]]", typeStr,
		"[[transaction_description]]", titleCase(tx.Description),
		"[[transaction_category]]", titleCase(tx.Category),
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
