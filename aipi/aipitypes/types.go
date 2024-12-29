package aipitypes

type AIPIResponse struct {
	Data string `json:"data"`
}

type AIPIRequest struct {
	SystemMessage string `json:"system_message"`
	UserMessage   string `json:"user_message"`
	Model         string `json:"model"`
}
