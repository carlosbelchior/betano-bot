package green365

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

const (
	BaseURL = "https://api-v2.green365.com.br/api/v2"
)

// Client handles all communication with the Green365 API.
type Client struct {
	baseURL  string
	resty    *resty.Client
	email    string
	password string
}

// NewClient initializes a new Green365 API client.
func NewClient() *Client {
	r := resty.New().
		SetBaseURL(BaseURL).
		SetTimeout(10 * time.Second).
		SetHeader("Content-Type", "application/json")

	return &Client{
		baseURL: BaseURL,
		resty:   r,
	}
}

// Authenticate logs in to the API and stores the credentials for automatic re-authentication.
func (c *Client) Authenticate(email, password string) error {
	c.email = email
	c.password = password
	return c.login()
}

func (c *Client) login() error {
	log.Info().Str("email", c.email).Msg("Attempting to authenticate with Green365")

	var result LoginResponse
	resp, err := c.resty.R().
		SetBody(LoginRequest{
			Email:    c.email,
			Password: c.password,
		}).
		SetResult(&result).
		Post("/auth/login")

	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	// Set the token for all future requests
	c.resty.SetAuthToken(result.Token)
	log.Debug().Msg("Authentication successful, token updated")
	return nil
}

// GetBotStats retrieves statistics for the specified bot.
func (c *Client) GetBotStats(botID string) (*BotStats, error) {
	var stats BotStats
	err := c.execute("GET", fmt.Sprintf("/bots/%s/stats", botID), nil, &stats)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetOpportunities retrieves the latest opportunities for the specified bot.
func (c *Client) GetOpportunities(botID string, limit int) (*OpportunitiesResponse, error) {
	var oppResp OpportunitiesResponse
	err := c.execute("GET", fmt.Sprintf("/bots/%s/opportunities", botID), map[string]string{
		"limit": fmt.Sprintf("%d", limit),
	}, &oppResp)
	if err != nil {
		return nil, err
	}
	return &oppResp, nil
}

// WatchOpportunities starts a polling loop to watch for new bot opportunities.
// It is a blocking call that respects the provided context.
func (c *Client) WatchOpportunities(ctx context.Context, botID string, interval time.Duration, callback func(Opportunity)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			resp, err := c.GetOpportunities(botID, 10)
			if err != nil {
				log.Error().Err(err).Msg("Failed to fetch opportunities")
				continue
			}

			for _, item := range resp.Items {
				callback(item)
			}
		}
	}
}

// execute performs an HTTP request with automatic re-authentication if a 403 Forbidden is encountered.
func (c *Client) execute(method, path string, queryParams map[string]string, result interface{}) error {
	run := func() (*resty.Response, error) {
		return c.resty.R().
			SetQueryParams(queryParams).
			SetResult(result).
			Execute(method, path)
	}

	resp, err := run()
	if err != nil {
		return fmt.Errorf("request %s %s failed: %w", method, path, err)
	}

	// Handle 403 by re-logging once and retrying
	if resp.StatusCode() == http.StatusForbidden && c.email != "" {
		log.Warn().Msg("Received 403 Forbidden. Attempting to re-authenticate...")
		if err := c.login(); err != nil {
			return fmt.Errorf("automatic re-authentication failed: %w", err)
		}

		// Retry the request
		resp, err = run()
		if err != nil {
			return fmt.Errorf("retry %s %s failed: %w", method, path, err)
		}
	}

	if resp.IsError() {
		return fmt.Errorf("request %s %s failed with status %d: %s", method, path, resp.StatusCode(), resp.String())
	}

	return nil
}
