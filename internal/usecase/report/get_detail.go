package report

import (
	"context"
	"regexp"
	"strconv"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type GetDetail struct {
	transactionRepo TransactionRepository
	userRepo        UserRepository

	geminiClient GeminiClient
}

func NewGetDetail(
	transactionRepo TransactionRepository,
	userRepo UserRepository,

	geminiClient GeminiClient,
) *GetDetail {
	return &GetDetail{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		geminiClient:    geminiClient,
	}
}

type (
	DetailResult struct {
		Transactions []*domain.Transaction
		Page         DetailPage
	}

	DetailPage struct {
		TotalPage   int
		CurrentPage int
		Period      domain.TransactionPeriod
		Start       int
		End         int
		TotalCount  int
	}
)

const (
	defaultGetDetailLimit  = 2
	defaultGetDetailOffset = 0

	quotedPeriodDateLayout = "02/01/06"
)

var (
	commandNextPage = []string{"next", "n", "lanjut", "lanjutkan", "selanjutnya", "berikutnya"}
	commandPrevPage = []string{"prev", "b", "previous", "back", "sebelumnya", "sebelum", "kembali"}

	quotedTextGetCurrentPageRegex   = regexp.MustCompile(`PAGE\s+(\d+)/(\d+)`)
	quotedTextGetCurrentPeriodRegex = regexp.MustCompile(`Period:\s*\*?(\d{2}/\d{2}/\d{2})\s*[-–—]\s*(\d{2}/\d{2}/\d{2})`)
	firstNumberRegex                = regexp.MustCompile(`\d+`)
)

func (uc *GetDetail) Execute(ctx context.Context, rawText, quotedText string, user *domain.User, isFirstPage bool) (*DetailResult, error) {
	// validate user
	isValidUser, userID := ValidateUser(ctx, user)
	if !isValidUser {
		return nil, util.Wrap(constant.ErrInvalidUser, "invalid user")
	}

	if isFirstPage {
		return uc.GetDetailFirstPage(ctx, rawText, userID)
	} else {
		return uc.GetDetailSpecificPage(ctx, rawText, quotedText, userID)
	}
}

func (uc *GetDetail) GetDetailFirstPage(ctx context.Context, rawText string, userID int64) (*DetailResult, error) {
	// parse message
	message, err := uc.geminiClient.ParseReportDetailMessage(ctx, rawText)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "parse message failed", err)
	}

	if !message.Valid || message.ReportMessage == nil {
		return nil, util.Wrap(constant.ErrInvalidRequest, "message is not a contain report information")
	}

	// generate transaction period
	transactionPeriod := generateTransactionPeriod(message.ReportMessage)
	if transactionPeriod == nil {
		return nil, util.Wrap(constant.ErrUnclearReportPeriod, "invalid transaction period")
	}
	if !ValidateTransactionRange(transactionPeriod) {
		return nil, util.Wrap(constant.ErrInvalidReportPeriodRange, "invalid transaction range")
	}

	// get list of transactions
	trarnsactions, totalCount, err := uc.transactionRepo.GetListByPeriod(
		ctx,
		userID,
		transactionPeriod.Start,
		transactionPeriod.End,
		defaultGetDetailLimit,
		defaultGetDetailOffset,
	)

	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "get transaction period failed", err)
	}

	if len(trarnsactions) == 0 {
		return nil, util.Wrap(constant.ErrReportNotFound, "transaction not found")
	}

	return detailed(trarnsactions, transactionPeriod, totalCount, 0), nil
}

func (uc *GetDetail) GetDetailSpecificPage(ctx context.Context, rawText, quotedText string, userID int64) (*DetailResult, error) {
	// parsed quoted message
	reportMessage := parsedQuotedText(rawText, quotedText)
	if reportMessage == nil {
		return nil, util.Wrap(constant.ErrInvalidRequest, "message is not a contain report information")
	}

	// generate transaction period
	transactionPeriod := generateTransactionPeriod(reportMessage)
	if transactionPeriod == nil {
		return nil, util.Wrap(constant.ErrUnclearReportPeriod, "invalid transaction period")
	}
	if !ValidateTransactionRange(transactionPeriod) {
		return nil, util.Wrap(constant.ErrInvalidReportPeriodRange, "invalid transaction range")
	}
	// get list of transactions by specific page
	trarnsactions, totalCount, err := uc.transactionRepo.GetListByPeriod(
		ctx,
		userID,
		transactionPeriod.Start,
		transactionPeriod.End,
		defaultGetDetailLimit,
		reportMessage.Offset,
	)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "get transaction period failedd", err)
	}

	if len(trarnsactions) == 0 {
		return nil, util.Wrap(constant.ErrReportNotFound, "transaction not found")
	}

	return detailed(trarnsactions, transactionPeriod, totalCount, reportMessage.Offset), nil
}

func detailed(transactions []*domain.Transaction, transactionPeriod *domain.TransactionPeriod, totalCount, offset int) *DetailResult {
	var (
		totalPage   int
		currentPage int
		start       int
		end         int
	)

	if totalCount > 0 {
		totalPage = totalCount / defaultGetDetailLimit
		if totalCount%defaultGetDetailLimit > 0 {
			totalPage += 1
		}
	}

	currentPage = offset/defaultGetDetailLimit + 1
	start = offset + 1
	end = start + len(transactions) - 1

	return &DetailResult{
		Transactions: transactions,
		Page: DetailPage{
			TotalPage:   totalPage,
			CurrentPage: currentPage,
			Period: domain.TransactionPeriod{
				Start: transactionPeriod.Start,
				End:   transactionPeriod.End,
			},
			Start:      start,
			End:        end,
			TotalCount: totalCount,
		},
	}
}

func parsedQuotedText(rawText, quotedText string) *domain.ParsedReportMessage {
	var (
		err error

		currentPage int64
		startDate   string
		endDate     string
	)

	// get current page
	// sample: "... PAGE 1/3 ..."
	// parsed: [PAGE][1][3]
	pageMessages := quotedTextGetCurrentPageRegex.FindStringSubmatch(quotedText)
	if len(pageMessages) != 3 {
		util.LogWarn("parsed pages failed", nil)
		return nil
	}

	if currentPage, err = strconv.ParseInt(pageMessages[1], 10, 32); err != nil {
		util.LogWarn("parsed current page failed", err)
		return nil
	}

	// get current period
	// sample: "... Period: 01 Sep 2026 – 30 Sep 2026 ..."
	// parsed: [Period][01 Sep 2026][30 Sep 2026]
	periodMessages := quotedTextGetCurrentPeriodRegex.FindStringSubmatch(quotedText)
	if len(periodMessages) != 3 {
		util.LogWarn("parsed period failed", nil)
		return nil
	}

	startDate, endDate = periodMessages[1], periodMessages[2]

	// get new offset
	newOffset := getNewOffset(rawText, int(currentPage), defaultGetDetailLimit)

	return &domain.ParsedReportMessage{
		StartDate:  startDate,
		EndDate:    endDate,
		Offset:     newOffset,
		DateLayout: quotedPeriodDateLayout,
	}
}

func getNewOffset(rawText string, currentPage, limit int) int {
	var newPage int
	if getPageStr := firstNumberRegex.FindString(rawText); getPageStr != "" {
		newPage, _ = strconv.Atoi(getPageStr)
	}

	if newPage > 0 {
		return (newPage - 1) * limit
	}

	newPage = currentPage
	switch {
	case util.ContainsAnyWord(rawText, commandNextPage):
		newPage += 1
	case util.ContainsAnyWord(rawText, commandPrevPage):
		newPage -= 1
	default:
	}

	return (newPage - 1) * limit
}
