package render

import (
	"context"
	"whatsupbot/internal/util"
)

type MCPTokenResponseSuccess struct {
	ctx    context.Context
	mcpURL string
	token  string
}

func NewMCPTokenResponseSuccess(
	ctx context.Context,
	mcpURL string,
	token string,
) *MCPTokenResponseSuccess {
	return &MCPTokenResponseSuccess{
		ctx:    ctx,
		mcpURL: mcpURL,
		token:  token,
	}
}

func (r *MCPTokenResponseSuccess) RenderSuccess() []Reply {
	// render mcp token template
	message, err := util.Render(templateNameMCPToken, TemplateDataMCPToken{MCPURL: r.mcpURL})
	if err != nil {
		util.LogError("Render mcp token template failed", err)
		return []Reply{}
	}

	// the token goes in its own message so it can be copied with one tap
	return []Reply{
		{Text: message},
		{Text: r.token},
	}
}
