package gemini

import "whatsup-bot/internal/domain"

type geminiResponse struct {
	Valid       bool   `json:"valid"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type geminiAmendResponse struct {
	Action      string `json:"action"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

func categoryEnum() []string {
	values := make([]string, len(domain.AllCategories))
	for i, c := range domain.AllCategories {
		values[i] = c.String()
	}
	return values
}
