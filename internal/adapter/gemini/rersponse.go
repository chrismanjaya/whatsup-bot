package gemini

type geminiResponse struct {
	Valid       bool   `json:"valid"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Date        string `json:"date"`
}
