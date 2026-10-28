package mcp

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"whatsupbot/internal/constant"
	"whatsupbot/internal/infra/service"
	"whatsupbot/internal/util"

	"github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "whatsup-bot"
	serverVersion = "1.0.0"

	endpointPath = "/mcp"

	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

type (
	Handler struct {
		server *http.Server
	}

	Config struct {
		Addr string
	}
)

func NewHandler(
	cfg Config,
	mcpService *service.MCP,
) *Handler {
	mcpServer := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	tools := &transactionTools{mcpService: mcpService}
	tools.register(mcpServer)

	mcpHandler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return mcpServer
	}, &mcpsdk.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.Handle(endpointPath, auth.RequireBearerToken(tokenVerifier(mcpService), nil)(mcpHandler))

	return &Handler{
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           mux,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

// Serve blocks until the server stops. It returns nil after a graceful Shutdown.
func (h *Handler) Serve() error {
	if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return util.Wrap(err, "serve mcp failed")
	}
	return nil
}

func (h *Handler) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	return h.server.Shutdown(ctx)
}

// tokenVerifier resolves a user's MCP token (issued over WhatsApp) to that user. Tools read the user ID back with userIDFromContext.
func tokenVerifier(mcpService *service.MCP) auth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		user, err := mcpService.Authenticate(ctx, token)
		if errors.Is(err, constant.ErrInvalidMCPToken) {
			return nil, auth.ErrInvalidToken
		}
		if err != nil {
			util.LogError("authenticate mcp token failed", err)
			return nil, errors.New("internal error")
		}

		return &auth.TokenInfo{
			UserID: strconv.FormatInt(user.ID, 10),
			// a token is valid until the user replaces it; the expiration only satisfies the sdk
			Expiration: time.Now().Add(time.Hour),
		}, nil
	}
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	tokenInfo := auth.TokenInfoFromContext(ctx)
	if tokenInfo == nil {
		return 0, false
	}

	userID, err := strconv.ParseInt(tokenInfo.UserID, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}

	return userID, true
}
