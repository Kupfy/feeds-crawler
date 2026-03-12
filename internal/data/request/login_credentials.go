package request

type LoginCredentials struct {
	LoginURL         string            `json:"loginUrl,omitempty"`
	Username         string            `json:"username,omitempty"`
	Email            string            `json:"email,omitempty"`
	Password         string            `json:"password,omitempty"`
	SuccessURL       string            `json:"successUrl,omitempty"`
	TokenSelector    string            `json:"tokenSelector,omitempty"`
	AdditionalFields map[string]string `json:"additionalFields,omitempty"`
}
