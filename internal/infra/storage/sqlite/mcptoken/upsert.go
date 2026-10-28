package mcptoken

import (
	"context"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

// a user holds at most one token: issuing a new one replaces (and so revokes) the old one
const upsertQuery = `
		INSERT INTO mcp_tokens (
			user_id,
			token_hash
		) VALUES (?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			token_hash = excluded.token_hash,
			created_at = CURRENT_TIMESTAMP
`

func (r *Repository) Upsert(ctx context.Context, token *domain.MCPToken) error {
	_, err := r.db.ExecContext(ctx, upsertQuery,
		token.UserID,
		token.TokenHash,
	)
	if err != nil {
		return util.Wrap(err, "upsert mcp token failed")
	}

	return nil
}
