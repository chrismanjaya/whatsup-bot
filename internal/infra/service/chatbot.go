package service

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	groupuc "whatsupbot/internal/usecase/group"
	integrationuc "whatsupbot/internal/usecase/integration"
	personauc "whatsupbot/internal/usecase/persona"
	predictionuc "whatsupbot/internal/usecase/prediction"
	"whatsupbot/internal/usecase/render"
	reportuc "whatsupbot/internal/usecase/report"
	transactionuc "whatsupbot/internal/usecase/transaction"
	useruc "whatsupbot/internal/usecase/user"
	"whatsupbot/internal/util"
)

type Chatbot struct {
	// group usecase
	validateGroup *groupuc.Validate

	// transaction usecase
	recordTransaction *transactionuc.Record
	amendTransaction  *transactionuc.Amend
	splitTransaction  *transactionuc.Split

	// user usecase
	registerUser *useruc.Register
	validateUser *useruc.Validate

	// report usecase
	getDetailReport  *reportuc.GetDetail
	getSummaryReport *reportuc.GetSummary

	// integration usecase
	generateMCPToken *integrationuc.GenerateMCPToken

	// persona usecase
	picker *personauc.Picker
}

func NewChatbot(
	validateGroup *groupuc.Validate,
	recordTransaction *transactionuc.Record,
	amendTransaction *transactionuc.Amend,
	splitTransaction *transactionuc.Split,
	registerUser *useruc.Register,
	validateUser *useruc.Validate,
	getDetailReport *reportuc.GetDetail,
	getSummaryReport *reportuc.GetSummary,
	generateMCPToken *integrationuc.GenerateMCPToken,
	picker *personauc.Picker,
) *Chatbot {
	return &Chatbot{
		validateGroup:     validateGroup,
		recordTransaction: recordTransaction,
		amendTransaction:  amendTransaction,
		splitTransaction:  splitTransaction,
		registerUser:      registerUser,
		validateUser:      validateUser,
		getDetailReport:   getDetailReport,
		getSummaryReport:  getSummaryReport,
		generateMCPToken:  generateMCPToken,
		picker:            picker,
	}
}

type IncomingMessage struct {
	SenderJID  string
	Text       string
	StanzaID   string
	QuotedText string
	IsReply    bool
	IsGroup    bool
	GroupName  string
}

var (
	successCode        *constant.ErrCode
	undefinedErrorCode *constant.ErrCode
)

func (s *Chatbot) ProcessIncomingMessage(ctx context.Context, message *IncomingMessage) []render.Reply {
	util.LogInfo("message received", message)

	var (
		err    error
		action predictionuc.Action
		group  *domain.Group
		user   *domain.User
	)

	// get action
	action = predictionuc.PredictAction(
		ctx,
		message.Text,
		message.QuotedText,
		message.IsGroup,
		message.IsReply,
	)

	// validate user
	if action != predictionuc.ACTION_REGISTRATION {
		if user, err = s.validateUser.Execute(ctx, message.SenderJID); err != nil {
			util.LogError("validate user failed", err, "chat_jid", message.SenderJID)

			return s.handleErrorEvent(ctx, message.SenderJID, constant.ErrUserNotRegister)
		}
	}

	// validate group
	if message.IsGroup {
		if group, err = s.validateGroup.Execute(ctx, message.SenderJID, message.GroupName); err != nil {
			util.LogError("validate group failed", err, "chat_jid", message.SenderJID)

			return s.handleErrorEvent(ctx, message.SenderJID, constant.ErrInvalidGroup)
		}
	}

	if message.IsGroup && action == predictionuc.ACTION_SPLIT {
		return s.SplitTransaction(ctx, message.SenderJID, group)
	}

	// process action
	switch action {
	case predictionuc.ACTION_REGISTRATION:
		return s.RegisterUser(ctx, message.SenderJID, message.Text)

	case predictionuc.ACTION_SUMMARY:
		return s.GetSummaryReport(ctx, message.SenderJID, message.Text, user)

	case predictionuc.ACTION_DETAIL:
		return s.GetDetailReport(ctx, message.SenderJID, message.Text, message.QuotedText, user, true)

	case predictionuc.ACTION_DETAIL_PAGE:
		return s.GetDetailReport(ctx, message.SenderJID, message.Text, message.QuotedText, user, false)

	case predictionuc.ACTION_AMEND:
		return s.AmendTransaction(ctx, message.SenderJID, message.Text, user, group, message.StanzaID)

	case predictionuc.ACTION_RECORD:
		return s.RecordTransaction(ctx, message.SenderJID, message.Text, user, group)

	case predictionuc.ACTION_MCP:
		return s.GenerateMCPToken(ctx, message.SenderJID, user, message.IsGroup)

	default:
		return s.handleErrorEvent(ctx, message.SenderJID, constant.ErrInvalidRequest)

	}
}

func (s *Chatbot) RegisterUser(ctx context.Context, jid, rawText string) []render.Reply {
	registeredUser, err := s.registerUser.Execute(ctx, jid, rawText)
	if err == nil && registeredUser != nil {
		registerResponse := render.NewRegisterResponseSuccess(
			ctx,
			registeredUser,
		)
		return registerResponse.RenderSuccess()
	}

	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) SplitTransaction(ctx context.Context, jid string, group *domain.Group) []render.Reply {
	result, err := s.splitTransaction.Execute(ctx, group)
	if err == nil && result != nil {
		var memberShares []render.MemberShare
		for _, memberShare := range result.MemberShares {
			memberShares = append(memberShares, render.NewMemberShare(
				memberShare.Name,
				memberShare.Spent,
				memberShare.Owes,
			))
		}

		splitResponse := render.NewSplitResponseSuccess(
			ctx,
			memberShares,
			result.Total,
			result.Share,
		)
		return splitResponse.RenderSuccess()
	}

	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) GetSummaryReport(ctx context.Context, jid string, rawText string, user *domain.User) []render.Reply {
	result, err := s.getSummaryReport.Execute(ctx, rawText, user)
	if err == nil && result != nil {
		topExpenseCategory, _ := result.ExpenseInfo.GetTopTransaction()
		topIncomeCategory, _ := result.IncomeInfo.GetTopTransaction()

		topExpenseTransaction := result.ExpenseInfo.GetHighestTransaction()
		topIncomeTransaction := result.IncomeInfo.GetHighestTransaction()

		summaryExpense := render.NewSummaryInfo(
			result.ExpenseInfo.Total,
			result.ExpenseInfo.Count,
			topExpenseCategory,
			topExpenseTransaction,
		)

		summaryIncome := render.NewSummaryInfo(
			result.IncomeInfo.Total,
			result.IncomeInfo.Count,
			topIncomeCategory,
			topIncomeTransaction,
		)

		reportSummaryResponse := render.NewReportSummaryResponseSuccess(
			ctx,
			result.Transactions,
			result.TransactionPeriod,
			summaryExpense,
			summaryIncome,
		)
		return reportSummaryResponse.RenderSuccess()
	}
	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) GetDetailReport(ctx context.Context, jid string, rawText, quotedText string, user *domain.User, isFirstPage bool) []render.Reply {
	result, err := s.getDetailReport.Execute(ctx, rawText, quotedText, user, isFirstPage)
	if err == nil && result != nil {
		reportDetailResponse := render.NewReportDetailResponseSuccess(
			ctx,
			result.Transactions,
			result.Page.TotalPage,
			result.Page.CurrentPage,
			result.Page.Period,
			result.Page.Start,
			result.Page.End,
			result.Page.TotalCount,
		)
		return reportDetailResponse.RenderSuccess()
	}
	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) AmendTransaction(ctx context.Context, jid string, rawText string, user *domain.User, group *domain.Group, messageID string) []render.Reply {
	result, err := s.amendTransaction.Execute(ctx, rawText, user, group, messageID)
	if err == nil && result != nil {
		amendResponse := render.NewAmendResponseSuccess(
			ctx,
			result.Transaction,
			result.Outcome == transactionuc.AmendOutcomeUpdated,
			result.Outcome == transactionuc.AmendOutcomeDeleted,
		)
		return amendResponse.RenderSuccess()
	}
	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) RecordTransaction(ctx context.Context, jid string, rawText string, user *domain.User, group *domain.Group) []render.Reply {
	result, err := s.recordTransaction.Execute(ctx, rawText, user, group)
	if err == nil && result != nil {
		recordResponse := render.NewRecordResponseSuccess(
			ctx,
			result.Transaction,
			result.ModelQuip,
			result.IsFirstToday,
		)
		return recordResponse.RenderSuccess()
	}

	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) GenerateMCPToken(ctx context.Context, jid string, user *domain.User, isGroup bool) []render.Reply {
	result, err := s.generateMCPToken.Execute(ctx, user, isGroup)
	if err == nil && result != nil {
		mcpTokenResponse := render.NewMCPTokenResponseSuccess(
			ctx,
			result.MCPURL,
			result.Token,
		)
		return mcpTokenResponse.RenderSuccess()
	}

	return s.handleErrorEvent(ctx, jid, err)
}

func (s *Chatbot) UpdateMessageID(ctx context.Context, transactionID int64, messageID string) error {
	return s.recordTransaction.UpdateMessageID(ctx, transactionID, messageID)
}

func mappingResponseCode(err error) *constant.ErrCode {
	if err == nil {
		return successCode
	}

	code, isDefined := util.CodeOf(err)
	if !isDefined {
		return undefinedErrorCode
	}

	return code
}

func (s *Chatbot) handleErrorEvent(ctx context.Context, jid string, err error) []render.Reply {
	errorCode := mappingResponseCode(err)
	errorMessage := s.picker.Pick(jid, errorCode)
	errorResponse := render.NewResponseError(ctx, errorCode, errorMessage)
	return errorResponse.RenderError()
}
