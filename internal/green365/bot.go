package green365

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	PollInterval  = 2 * time.Second
	StatsInterval = 5 * time.Minute
)

// Bot centralizes the logic for interacting with the Green365 site.
type Bot struct {
	client         *Client
	email          string
	password       string
	botID          string
	opportunityIDs map[int]bool
	stopChan       chan struct{}
}

// NewBot initializes a new Green365 bot instance.
func NewBot(email, password, botID string) *Bot {
	return &Bot{
		client:         NewClient(),
		email:          email,
		password:       password,
		botID:          botID,
		opportunityIDs: make(map[int]bool),
		stopChan:       make(chan struct{}),
	}
}

// Run starts the bot lifecycle.
// It returns an error if the initial authentication fails.
func (b *Bot) Run(ctx context.Context) error {
	// 1. Initial Authentication
	if err := b.client.Authenticate(b.email, b.password); err != nil {
		return fmt.Errorf("initial authentication failed: %w", err)
	}

	// 2. Initial Stats Fetch
	b.fetchAndPrintStats()

	// Periodic Stats Poller
	go func() {
		ticker := time.NewTicker(StatsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.fetchAndPrintStats()
			case <-b.stopChan:
				log.Info().Msg("Stats poller shutting down...")
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// 3. Watch Opportunities (Blocking)
	log.Info().Str("botID", b.botID).Msg("Starting opportunities watcher")

	// WatchOpportunities is a blocking call. We use a context to stop it if needed.
	return b.client.WatchOpportunities(ctx, b.botID, PollInterval, func(opp Opportunity) {
		if !b.opportunityIDs[opp.ID] {
			b.opportunityIDs[opp.ID] = true
			log.Info().
				Int("oppID", opp.ID).
				Str("time", time.Now().Format("15:04:05")).
				Msg("New bot opportunity found")
		}
	})
}

// Stop sends a signal to stop the bot's background polling.
func (b *Bot) Stop() {
	close(b.stopChan)
}

func (b *Bot) fetchAndPrintStats() {
	stats, err := b.client.GetBotStats(b.botID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch bot statistics")
		return
	}
	b.printStats(stats)
}

func (b *Bot) printStats(stats *BotStats) {
	// Colors
	blueBold := "\033[1;34m"
	green := "\033[32m"
	red := "\033[31m"
	reset := "\033[0m"

	winRateColor := green
	if stats.Overview.WinRate < 0 {
		winRateColor = red
	}

	profitColor := green
	if stats.Overview.Profit < 0 {
		profitColor = red
	}

	fmt.Printf("%s%s%s : %d (Tips) | %s%.2f%%%s | %sR$ %.2f%s\n",
		blueBold, stats.BotName, reset,
		stats.Overview.Opportunities,
		winRateColor, stats.Overview.WinRate, reset,
		profitColor, stats.Overview.Profit, reset,
	)
}
