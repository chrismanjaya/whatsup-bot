package mcptoken

import (
	"database/sql"
	"errors"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const mcpTokenColumns = `
	id,
	user_id,
	token_hash,
	created_at
`

func scanRow(row *sql.Row) (*domain.MCPToken, error) {
	var token domain.MCPToken

	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, util.Wrap(err, "scan mcp token")
	}

	return &token, nil
}
