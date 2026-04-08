package green365

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://api-v2.green365.com.br/api/v2"
)

// Green365 wraps the Client and provides higher-level operations
type Green365 struct {
	client *Client
}

// NewGreen365 creates a new instance of the central Green365 handler
func NewGreen365() *Green365 {
	return &Green365{
		client: NewClient(),
	}
}

// Start authenticates and returns an error if login fails
func (g *Green365) Start(email, password string) error {
	return g.client.Login(email, password)
}

// GetStats returns statistics for the given bot
func (g *Green365) GetStats(botID string) (*BotStats, error) {
	return g.client.GetBotStats(botID)
}

// WatchOpportunities polls for new opportunities and calls the callback for each new item found
func (g *Green365) WatchOpportunities(botID string, interval time.Duration, callback func(Opportunity)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := g.client.GetOpportunities(botID, 10)
		if err != nil {
			fmt.Printf("Error fetching opportunities: %v\n", err)
			continue
		}

		for _, item := range resp.Items {
			callback(item)
		}
	}
	return nil
}

// Client handles communication with the Green365 API
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
	Email      string
	Password   string
}

// NewClient creates a new Green365 client
func NewClient() *Client {
	return &Client{
		BaseURL: BaseURL,
		HTTPClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

// Login authenticates with the API and stores the Bearer token
func (c *Client) Login(email, password string) error {
	url := fmt.Sprintf("%s/auth/login", c.BaseURL)

	c.Email = email
	c.Password = password

	reqBody, _ := json.Marshal(LoginRequest{
		Email:    email,
		Password: password,
	})

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.Token = loginResp.Token
	return nil
}

// doRequest handles making HTTP requests with automatic re-login on 403
func (c *Client) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	makeReq := func() (*http.Request, error) {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, err
		}
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		return req, nil
	}

	req, err := makeReq()
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If 403, try to re-login and retry once
	if resp.StatusCode == http.StatusForbidden && c.Email != "" && c.Password != "" {
		resp.Body.Close()
		fmt.Println("Received 403 Forbidden. Attempting to re-login...")

		if err := c.Login(c.Email, c.Password); err != nil {
			return nil, fmt.Errorf("re-login failed after 403: %w", err)
		}

		// Re-create request for retry (in case body needs to be reset, though nil here)
		req, err = makeReq()
		if err != nil {
			return nil, err
		}

		return c.HTTPClient.Do(req)
	}

	return resp, nil
}

// GetBotStats retrieves statistics for a specific bot
func (c *Client) GetBotStats(botID string) (*BotStats, error) {
	url := fmt.Sprintf("%s/bots/%s/stats", c.BaseURL, botID)

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("stats request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stats failed with status: %s", resp.Status)
	}

	var stats BotStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &stats, nil
}

// GetOpportunities retrieves the latest opportunities for a specific bot
func (c *Client) GetOpportunities(botID string, limit int) (*OpportunitiesResponse, error) {
	url := fmt.Sprintf("%s/bots/%s/opportunities?limit=%d", c.BaseURL, botID, limit)

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("opportunities request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Debug - Ops error body: %s\n", string(bodyBytes))
		return nil, fmt.Errorf("opportunities failed with status: %s", resp.Status)
	}

	var oppResp OpportunitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&oppResp); err != nil {
		return nil, fmt.Errorf("failed to decode opportunities: %w", err)
	}

	return &oppResp, nil
}
