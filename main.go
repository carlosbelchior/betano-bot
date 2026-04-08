package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/carlosbelchior/betano-bot/internal/green365"
	"github.com/joho/godotenv"
)

const (
	PollInterval  = 2 * time.Second
	StatsInterval = 5 * time.Minute
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	email := os.Getenv("GREEN365_EMAIL")
	password := os.Getenv("GREEN365_PASSWORD")
	botID := os.Getenv("GREEN365_BOT_ID")

	if email == "" || password == "" || botID == "" {
		log.Fatal("GREEN365_EMAIL, GREEN365_PASSWORD and GREEN365_BOT_ID must be set in .env")
	}

	// Initialize Central Green365 Handler
	g := green365.NewGreen365()

	// 1. Auth Login
	fmt.Printf("Logging in as %s...\n", email)
	if err := g.Start(email, password); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Println("Login successful!")

	// 2. Fetch Stats initially and then every 5 minutes
	fetchAndPrintStats := func() {
		stats, err := g.GetStats(botID)
		if err != nil {
			log.Printf("Error fetching stats: %v", err)
			return
		}
		printStats(stats)
	}

	// Initial fetch
	fetchAndPrintStats()

	// Ticker for periodic stats
	go func() {
		ticker := time.NewTicker(StatsInterval)
		defer ticker.Stop()
		for range ticker.C {
			fetchAndPrintStats()
		}
	}()

	// 3. Watch Opportunities
	fmt.Println("Starting opportunities watching...")

	// Map to store unique IDs found (to demonstrate handling in main)
	opportunityIDs := make(map[int]bool)

	err := g.WatchOpportunities(botID, PollInterval, func(opp green365.Opportunity) {
		if !opportunityIDs[opp.ID] {
			opportunityIDs[opp.ID] = true
			fmt.Printf("[%s] New Opportunity Found: ID %d\n", time.Now().Format("15:04:05"), opp.ID)
		}
	})

	if err != nil {
		log.Fatalf("Watcher failed: %v", err)
	}
}

func printStats(stats *green365.BotStats) {
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
