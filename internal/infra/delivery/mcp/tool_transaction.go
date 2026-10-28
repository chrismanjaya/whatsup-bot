package mcp

import (
	"context"
	"errors"
	"strings"
	"time"

	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/infra/service"
	reportusecase "whatsupbot/internal/usecase/report"
	"whatsupbot/internal/util"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	toolGetTransactionList = "get_transaction_list"

	inputDateLayout = time.DateOnly

	typeIncome  = "income"
	typeExpense = "expense"
)

type (
	transactionTools struct {
		mcpService *service.MCP
	}

	GetTransactionListInput struct {
		StartDate string `json:"start_date" jsonschema:"start of the period (inclusive), format YYYY-MM-DD"`
		EndDate   string `json:"end_date" jsonschema:"end of the period (inclusive), format YYYY-MM-DD. The period may span at most 6 months"`
		Type      string `json:"type,omitempty" jsonschema:"optional filter: income or expense. Omit for all transactions"`
		Limit     int    `json:"limit,omitempty" jsonschema:"max transactions to return, default 50, max 200"`
		Offset    int    `json:"offset,omitempty" jsonschema:"number of transactions to skip, for paging. Default 0"`
	}

	GetTransactionListOutput struct {
		User         UserOutput          `json:"user"`
		StartDate    string              `json:"start_date"`
		EndDate      string              `json:"end_date"`
		Transactions []TransactionOutput `json:"transactions"`
		TotalCount   int                 `json:"total_count" jsonschema:"total transactions in the period, across all pages"`
		Limit        int                 `json:"limit"`
		Offset       int                 `json:"offset"`
		HasMore      bool                `json:"has_more" jsonschema:"true when more transactions remain; call again with offset = offset + limit"`
	}

	UserOutput struct {
		Name string `json:"name"`
	}

	TransactionOutput struct {
		Description string `json:"description"`
		Type        string `json:"type" jsonschema:"income or expense"`
		Amount      int64  `json:"amount" jsonschema:"amount in IDR"`
		Date        string `json:"date" jsonschema:"transaction date, format YYYY-MM-DD"`
		Category    string `json:"category"`
	}
)

func (t *transactionTools) register(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        toolGetTransactionList,
		Description: "List the authenticated user's income and expense transactions within a date period (max 6 months), ordered by date ascending. Results are paged.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, t.handleGetTransactionList)
}

func (t *transactionTools) handleGetTransactionList(ctx context.Context, _ *mcpsdk.CallToolRequest, input GetTransactionListInput) (*mcpsdk.CallToolResult, GetTransactionListOutput, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, GetTransactionListOutput{}, errors.New("unauthenticated")
	}

	req, err := toTransactionListRequest(userID, input)
	if err != nil {
		return nil, GetTransactionListOutput{}, err
	}

	result, err := t.mcpService.GetTransactionList(ctx, req)
	if err != nil {
		util.LogError("mcp get transaction list failed", err)
		return nil, GetTransactionListOutput{}, toToolError(err)
	}

	return nil, toGetTransactionListOutput(result), nil
}

func toTransactionListRequest(userID int64, input GetTransactionListInput) (reportusecase.TransactionListRequest, error) {
	start, err := time.ParseInLocation(inputDateLayout, input.StartDate, util.Jakarta)
	if err != nil {
		return reportusecase.TransactionListRequest{}, errors.New("start_date must use format YYYY-MM-DD")
	}

	end, err := time.ParseInLocation(inputDateLayout, input.EndDate, util.Jakarta)
	if err != nil {
		return reportusecase.TransactionListRequest{}, errors.New("end_date must use format YYYY-MM-DD")
	}

	var transactionType domain.TransactionType
	switch strings.ToLower(strings.TrimSpace(input.Type)) {
	case "":
	case typeIncome:
		transactionType = domain.Income
	case typeExpense:
		transactionType = domain.Expense
	default:
		return reportusecase.TransactionListRequest{}, errors.New("type must be income or expense")
	}

	return reportusecase.TransactionListRequest{
		UserID: userID,
		Period: domain.TransactionPeriod{Start: start, End: end},
		Type:   transactionType,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
}

func toGetTransactionListOutput(result *reportusecase.TransactionListResult) GetTransactionListOutput {
	transactions := make([]TransactionOutput, 0, len(result.Transactions))
	for _, transaction := range result.Transactions {
		transactions = append(transactions, TransactionOutput{
			Description: transaction.Description,
			Type:        toOutputType(transaction.Type),
			Amount:      transaction.Amount,
			Date:        transaction.TransactionDate.In(util.Jakarta).Format(inputDateLayout),
			Category:    string(transaction.Category),
		})
	}

	return GetTransactionListOutput{
		User:         UserOutput{Name: result.User.Name},
		StartDate:    result.Period.Start.Format(inputDateLayout),
		EndDate:      result.Period.End.Format(inputDateLayout),
		Transactions: transactions,
		TotalCount:   result.TotalCount,
		Limit:        result.Limit,
		Offset:       result.Offset,
		HasMore:      result.Offset+len(transactions) < result.TotalCount,
	}
}

func toOutputType(transactionType domain.TransactionType) string {
	if transactionType == domain.Income {
		return typeIncome
	}
	return typeExpense
}

// toToolError maps an application error to a message safe to show the calling AI tool; internal detail stays in logs.
func toToolError(err error) error {
	code, ok := util.CodeOf(err)
	if !ok {
		code = constant.ErrInternal
	}

	switch code {
	case constant.ErrUserNotRegister, constant.ErrInvalidUser:
		return errors.New("user is not registered")
	case constant.ErrInvalidReportPeriodRange:
		return errors.New("period must not exceed 6 months")
	case constant.ErrUnclearReportPeriod:
		return errors.New("end_date must not be before start_date")
	case constant.ErrInvalidRequest, constant.ErrInvalidTransactionType:
		return errors.New("invalid request")
	default:
		return errors.New("internal error, please try again later")
	}
}
