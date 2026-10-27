package render

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type RegisterResponseSuccess struct {
	ctx  context.Context
	user *domain.User
}

func NewRegisterResponseSuccess(
	ctx context.Context,
	user *domain.User,
) *RegisterResponseSuccess {
	return &RegisterResponseSuccess{
		ctx:  ctx,
		user: user,
	}
}

func (r *RegisterResponseSuccess) RenderSuccess() []Reply {
	// render registration template
	message, err := util.Render(templateNameRegistration, TemplateDataRegistration{Name: r.user.Name})
	if err != nil {
		util.LogError("Render registration template failed", err)
		return []Reply{}
	}

	metadata := Metadata{
		User: r.user,
	}

	return []Reply{
		{
			Text:     message,
			Metadata: metadata,
		},
	}
}
