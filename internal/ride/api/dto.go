package api

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type WSResponse struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Payload interface{} `json:"payload,omitempty"`
}
