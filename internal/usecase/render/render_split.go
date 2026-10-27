package render

import (
	"context"
	"strings"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

type (
	SplitResponseSuccess struct {
		ctx          context.Context
		memberShares []MemberShare
		total        int64
		share        int64
	}

	MemberShare struct {
		name  string
		spent int64
		owes  int64
	}
)

func NewSplitResponseSuccess(
	ctx context.Context,
	memberShares []MemberShare,
	total int64,
	share int64,
) *SplitResponseSuccess {
	return &SplitResponseSuccess{
		ctx:          ctx,
		memberShares: memberShares,
		total:        total,
		share:        share,
	}
}

func NewMemberShare(
	name string,
	spent int64,
	owes int64,
) MemberShare {
	return MemberShare{
		name:  name,
		spent: spent,
		owes:  owes,
	}
}

func (r *SplitResponseSuccess) RenderSuccess() []Reply {
	// render member shares
	var memberSharesMessage []string
	for _, memberShare := range r.memberShares {
		// render member share template
		message, err := util.Render(templateNameSplitMemberShares, TemplateDataSplitMemberShares{
			MemberName:      memberShare.name,
			AmountFormatted: util.FormatCurrency(memberShare.spent, "IDR", language.English),
		})
		if err != nil {
			util.LogError("Render member share template failed", err)
			return []Reply{}
		}
		memberSharesMessage = append(memberSharesMessage, message)
	}

	// render member owes
	var memberOwesMessage []string
	for _, memberShare := range r.memberShares {
		// render member owes template
		message, err := util.Render(templateNameSplitMemberOwes, TemplateDataSplitMemberOwes{
			MemberName:      memberShare.name,
			AmountFormatted: util.FormatCurrency(memberShare.owes, "IDR", language.English),
		})
		if err != nil {
			util.LogError("Render member owes template failed", err)
			return []Reply{}
		}
		memberOwesMessage = append(memberOwesMessage, message)
	}

	// render split transaction template
	message, err := util.Render(templateNameSplitTransaction, TemplateDataSplitTransaction{
		MemberShares:         strings.Join(memberSharesMessage, "\n"),
		MemberOwes:           strings.Join(memberOwesMessage, "\n"),
		TotalAmountFormatted: util.FormatCurrency(r.total, "IDR", language.English),
		ShareAmountFormatted: util.FormatCurrency(r.share, "IDR", language.English),
	})
	if err != nil {
		util.LogError("Render split template failed", err)
		return []Reply{}
	}
	return []Reply{
		{
			Text: message,
		},
	}
}
