package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/keshon/melodix/bot"
)

const envPath = "./.env"

func main() {
	if err := godotenv.Load(envPath); err != nil {
		log.Println("Error loading .env file")
	}

	if os.Getenv("DISCORD_TOKEN") == "" {
		log.Fatal("DISCORD_TOKEN is missing in environment variables")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("Discord token not found in environment variables")
	}

	cacheDir := "./cache"
	defer func() {
		if err := os.RemoveAll(cacheDir); err != nil {
			log.Printf("Failed to remove cache folder: %v", err)
		}
	}()

	bot, err := bot.NewBot(token)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}
	defer bot.Shutdown()

	if err := bot.Start(); err != nil {
		log.Fatal("Failed to start bot:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Fallback for local development
	}

	// Simple handler that returns 200 OK
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Bot is running"))
	})

	// Run the HTTP server in a goroutine so it doesn't block the rest of the code
	go func() {
		fmt.Printf("Web server listening on port %s\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("Failed to start HTTP server:", err)
		}
	}()

	// ---------------------------------------------------------

	fmt.Println("Bot is running. Press CTRL-C to exit.")
	
	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	
	fmt.Println("Shutting down...")
}
