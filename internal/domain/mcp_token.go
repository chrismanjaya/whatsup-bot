package domain

import "time"

type (
	// MCPToken grants an external AI tool read access to one user's data through the MCP server. Only the hash of the token is stored; the plain token is shown to the user once.
	MCPToken struct {
		ID        int64
		UserID    int64
		TokenHash string
		CreatedAt time.Time
	}
)
