package mcp

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"whatsupbot/internal/domain"
	"whatsupbot/internal/infra/service"
	"whatsupbot/internal/infra/storage/sqlite"
	mcptokenrepo "whatsupbot/internal/infra/storage/sqlite/mcptoken"
	transactionrepo "whatsupbot/internal/infra/storage/sqlite/transaction"
	userrepo "whatsupbot/internal/infra/storage/sqlite/user"
	integrationuc "whatsupbot/internal/usecase/integration"
	reportuc "whatsupbot/internal/usecase/report"
)

// newTestServer serves the mcp handler on a loopback address, like production behind Caddy, and returns its URL plus a valid token.
func newTestServer(t *testing.T) (string, string) {
	t.Helper()
	ctx := context.Background()

	client, err := sqlite.NewClient(ctx, sqlite.Config{DBPath: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatalf("init sqlite: %v", err)
	}
	db := client.Client()

	if _, err := db.Exec(`INSERT INTO users (jid, name, email) VALUES ('a', 'Budi', 'budi@mail.com'), ('b', 'Ani', 'ani@mail.com')`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO transactions (user_id, description, type, amount, category, transaction_date) VALUES
		(1, 'kopi', 'DB', 25000, 'food', '2026-09-10 08:00:00+07:00'),
		(2, 'ani only', 'DB', 1000, 'other', '2026-09-11 08:00:00+07:00')`); err != nil {
		t.Fatalf("seed transactions: %v", err)
	}

	userRepo, transactionRepo, mcpTokenRepo := userrepo.NewRepo(db), transactionrepo.NewRepo(db), mcptokenrepo.NewRepo(db)
	integrationModule := integrationuc.NewModule(mcpTokenRepo, userRepo, "")
	reportModule := reportuc.NewModule(transactionRepo, userRepo, nil)

	generated, err := integrationModule.GenerateMCPTokenUsecase().Execute(ctx, &domain.User{ID: 1}, false)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := NewHandler(Config{}, service.NewMCP(
		integrationModule.AuthenticateMCPTokenUsecase(),
		reportModule.GetTransactionListUsecase(),
	))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &http.Server{Handler: handler.server.Handler}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })

	return "http://" + listener.Addr().String() + endpointPath, generated.Token
}

func callTool(t *testing.T, url, host, token string) (int, string) {
	t.Helper()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_transaction_list","arguments":{"start_date":"2026-09-01","end_date":"2026-09-30"}}}`
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody)
}

func TestHandler(t *testing.T) {
	url, token := newTestServer(t)

	t.Run("proxied request with public host is served", func(t *testing.T) {
		status, body := callTool(t, url, "mcp.example.com", token)
		if status != http.StatusOK {
			t.Fatalf("status = %d, body = %s; want 200", status, body)
		}
		if !strings.Contains(body, "kopi") {
			t.Fatalf("body = %s; want the user's transaction", body)
		}
		if strings.Contains(body, "ani only") {
			t.Fatalf("body = %s; leaked another user's transaction", body)
		}
	})

	t.Run("missing token is rejected", func(t *testing.T) {
		if status, _ := callTool(t, url, "mcp.example.com", ""); status != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", status)
		}
	})

	t.Run("unknown token is rejected", func(t *testing.T) {
		if status, _ := callTool(t, url, "mcp.example.com", "wsb_unknown"); status != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", status)
		}
	})
}
