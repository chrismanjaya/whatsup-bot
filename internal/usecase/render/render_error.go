package render

import (
	"context"
	"whatsupbot/internal/constant"
)

type ResponseError struct {
	ctx          context.Context
	errCode      *constant.ErrCode
	errorMessage string
}

func NewResponseError(
	ctx context.Context,
	errCode *constant.ErrCode,
	errorMessage string,
) *ResponseError {
	return &ResponseError{
		ctx:          ctx,
		errCode:      errCode,
		errorMessage: errorMessage,
	}
}

func (r *ResponseError) RenderError() []Reply {
	return []Reply{{Text: r.errorMessage}}
}
