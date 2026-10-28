package render

import (
	"context"
	"strings"
	"unicode/utf8"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

const (
	ReactionPending = "⏳"
	ReactionSuccess = "✅"
	ReactionFailed  = "⚠️"

	// maxFailedTextRunes caps how much of a failed line is echoed back for a multi-line message.
	maxFailedTextRunes = 40
)

type (
	// RecordLineResult is the outcome of one line of a record message: Transaction when saved, otherwise a Frankie-style ErrorMessage from the persona pool (its emoji stands in for the error code).
	RecordLineResult struct {
		Text         string
		Transaction  *domain.Transaction
		Quip         string
		ErrorMessage string
	}

	RecordResponse struct {
		ctx   context.Context
		lines []RecordLineResult
	}
)

func NewRecordResponse(
	ctx context.Context,
	lines []RecordLineResult,
) *RecordResponse {
	return &RecordResponse{
		ctx:   ctx,
		lines: lines,
	}
}

// RenderRecordAccepted is sent right after a record message is queued: a pending reaction, plus a canned ack line when one is given.
func RenderRecordAccepted(ack string) []Reply {
	replies := []Reply{{Reaction: ReactionPending}}
	if ack != "" {
		replies = append(replies, Reply{Text: ack})
	}
	return replies
}

// Render replaces the pending reaction (✅ when every line saved, ⚠️ otherwise), then quotes the original message once per line: the transaction confirmation when saved, the Frankie error when not.
func (r *RecordResponse) Render() []Reply {
	var (
		reaction = ReactionSuccess
		replies  = make([]Reply, 0, len(r.lines)+1)
		isMulti  = len(r.lines) > 1
	)

	for _, line := range r.lines {
		if line.Transaction == nil {
			reaction = ReactionFailed
		}
		if reply, ok := r.renderLine(line, isMulti); ok {
			replies = append(replies, reply)
		}
	}

	return append([]Reply{{Reaction: reaction}}, replies...)
}

func (r *RecordResponse) renderLine(line RecordLineResult, isMulti bool) (Reply, bool) {
	if line.Transaction == nil {
		if line.ErrorMessage == "" {
			return Reply{}, false
		}

		// the quote shows the whole message, so name the failed line when there are several
		message := line.ErrorMessage
		if isMulti {
			message = "_\"" + truncateRunes(line.Text, maxFailedTextRunes) + "\"_\n" + message
		}
		return Reply{Text: message, IsQuoted: true}, true
	}

	// render record template
	message, err := util.Render(templateNameTransaction, TemplateDataTransaction{
		Type:            strings.ToUpper(convertTransactionType(line.Transaction.Type)),
		Description:     line.Transaction.Description,
		Category:        line.Transaction.Category.String(),
		AmountFormatted: util.FormatCurrency(line.Transaction.Amount, "IDR", language.English),
		DateFormatted:   TransactionDateFormatted(line.Transaction.TransactionDate),
	})
	if err != nil {
		util.LogError("Render transaction template failed", err)
		return Reply{}, false
	}

	return Reply{
		Text:     appendQuip(message, line.Quip),
		IsQuoted: true,
		Metadata: Metadata{
			Transaction: line.Transaction,
		},
	}, true
}

// appendQuip adds Frankie's quip as the last line of message, rendered through the gimmick template.
func appendQuip(message, quip string) string {
	if quip == "" {
		return message
	}

	gimmickMessage, err := util.Render(templateNameGimmick, TemplateDataGimmick{Quip: quip})
	if err != nil || gimmickMessage == "" {
		return message
	}

	return message + "\n\n" + gimmickMessage
}

func truncateRunes(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit-1]) + "…"
}
