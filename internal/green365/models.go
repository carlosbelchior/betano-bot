package green365

import "time"

// LoginRequest defines the structure for authentication
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse defines the structure for the authentication response
type LoginResponse struct {
	Token string `json:"token"`
}

// BotStats defines the structure for bot statistics response
type BotStats struct {
	BotName   string `json:"botName"`
	Stake     int    `json:"stake"`
	Status    string `json:"status"`
	Overview  struct {
		Opportunities int     `json:"opportunities"`
		Unit          float64 `json:"unit"`
		WinRate       float64 `json:"winRate"`
		Roi           float64 `json:"roi"`
		Profit        float64 `json:"profit"`
		AverageOdd    float64 `json:"averageOdd"`
	} `json:"overview"`
	Results []struct {
		Type       string  `json:"type"`
		Total      int     `json:"total"`
		Percentage float64 `json:"percentage"`
	} `json:"results"`
}

// Opportunity defines a single item in the opportunities array
type Opportunity struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	// Use map for unknown fields to keep it flexible
	Data map[string]interface{} `json:"data"`
}

// OpportunitiesResponse defines the structure for the opportunities response
type OpportunitiesResponse struct {
	Items      []Opportunity `json:"items"`
	TotalCount int           `json:"total_count"`
}
