package aipitypes

type AIPIResponse struct {
	Data string `json:"data"`
}

type ResponseFormat string

const AIPI_RESPONSE_FORMAT_JSON = "json_object"
const AIPI_RESPONSE_FORMAT_TEXT = "text"

type AIPIRequest struct {
	SystemMessage  string `json:"system_message"`
	UserMessage    string `json:"user_message"`
	Model          string `json:"model"`
	IdUser         uint   `json:"id_user"`
	ResponseFormat string `json:"response_format"`
}

type EmbeddingRequest struct {
	Input          any
	Model          string
	EncodingFormat string
	Dimensions     int
}
