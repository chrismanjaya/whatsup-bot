package prediction

import (
	"context"

	"whatsupbot/internal/util"
)

const (
	ACTION_UNKNOWN      = 0
	ACTION_RECORD       = 1
	ACTION_AMEND        = 2
	ACTION_SUMMARY      = 3
	ACTION_DETAIL       = 4
	ACTION_DETAIL_PAGE  = 5
	ACTION_REGISTRATION = 6
	ACTION_SPLIT        = 7
	ACTION_MCP          = 8
)

type Action int32

var (
	summaryKeywords      []string = []string{"summary", "summarize", "summarise", "report", "rangkum", "ringkas", "laporan", "rekap"}
	detalKeywords        []string = []string{"transaksi", "transaction", "riwayat", "history", "daftar", "list"}
	registrationKeywords []string = []string{"register"}
	splitKeywords        []string = []string{"split"}
	mcpKeywords          []string = []string{"mcp"}

	replyDetailKeyword []string = []string{"PAGE"}
	replyAmendKeyword  []string = []string{"EXPENSE", "INCOME"}
)

var (
	MappingAction = func(ctx context.Context, rawMessage string, isGroup bool) Action {
		switch {
		case util.ContainsAnyWord(rawMessage, registrationKeywords):
			return ACTION_REGISTRATION
		case util.ContainsAnyWord(rawMessage, mcpKeywords):
			return ACTION_MCP
		case isGroup && util.ContainsAnyWord(rawMessage, splitKeywords):
			return ACTION_SPLIT
		case util.ContainsAnyWord(rawMessage, summaryKeywords):
			return ACTION_SUMMARY
		case util.ContainsAnyWord(rawMessage, detalKeywords):
			return ACTION_DETAIL
		default:
			return ACTION_RECORD
		}
	}

	MappingReplyAction = func(ctx context.Context, quotedText string) Action {
		switch {
		case util.ContainsAnyWord(quotedText, replyDetailKeyword):
			return ACTION_DETAIL_PAGE
		case util.ContainsAnyWord(quotedText, replyAmendKeyword):
			return ACTION_AMEND
		default:
			return ACTION_UNKNOWN
		}
	}
)

func PredictAction(ctx context.Context, rawMessage, quotedText string, isGroup, isReply bool) Action {
	if isReply {
		return MappingReplyAction(ctx, quotedText)
	} else {
		return MappingAction(ctx, rawMessage, isGroup)
	}
}
