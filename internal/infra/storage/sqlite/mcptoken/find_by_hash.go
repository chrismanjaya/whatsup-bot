package mcptoken

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const findByHashQuery = `
		SELECT ` + mcpTokenColumns + ` FROM mcp_tokens
		WHERE
			token_hash = ?
	`

func (r *Repository) FindByHash(ctx context.Context, tokenHash string) (*domain.MCPToken, error) {
	row := r.db.QueryRowContext(ctx, findByHashQuery, tokenHash)
	result, err := scanRow(row)
	if err != nil {
		return nil, util.Wrap(err, "query mcp token failed")
	}

	return result, nil
}
