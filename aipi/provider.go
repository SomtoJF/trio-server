package aipi

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

type AIPIRequest struct {
	SystemMessage string `json:"system_message"`
	UserMessage   string `json:"user_message"`
}

func (p *Provider) GetResponse(request AIPIRequest) (string, error) {
	return "", nil
}

func (p *Provider) GetResponseAsync(request AIPIRequest) (string, error) {
	return "", nil
}
