package integration

type Module struct {
	generateMCPToken     *GenerateMCPToken
	authenticateMCPToken *AuthenticateMCPToken
}

func NewModule(
	mcpTokenRepo MCPTokenRepository,
	userRepo UserRepository,
	mcpURL string,
) *Module {
	return &Module{
		generateMCPToken:     NewGenerateMCPToken(mcpTokenRepo, mcpURL),
		authenticateMCPToken: NewAuthenticateMCPToken(mcpTokenRepo, userRepo),
	}
}

func (m *Module) GenerateMCPTokenUsecase() *GenerateMCPToken {
	return m.generateMCPToken
}

func (m *Module) AuthenticateMCPTokenUsecase() *AuthenticateMCPToken {
	return m.authenticateMCPToken
}
