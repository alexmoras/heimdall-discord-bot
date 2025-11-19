package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting Heimdall...")

	// Load configuration
	log.Println("Loading configuration...")
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	log.Println("✓ Configuration loaded")

	// Initialize logging system with config
	InitLogger(config.Server.LogLevel)
	LogDebug("Log level: %s", GetLogLevel())
	LogDebug("Loaded config: Guild=%s, Port=%d, Teams=%d", config.Discord.GuildID, config.Server.Port, len(config.Teams))

	// Initialize database
	log.Println("Initializing database...")

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Error creating data directory: %v", err)
	}

	db, err := NewDatabase("data/heimdall.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Database initialized")
	LogDebug("Database file: data/heimdall.db")

	// Initialize email service
	log.Println("Initializing email service...")
	emailService := NewEmailService(config)
	log.Println("✓ Email service ready")
	LogDebug("SMTP: %s:%d", config.Email.SMTPHost, config.Email.SMTPPort)

	// Initialize Discord bot
	log.Println("Initializing Discord bot...")
	bot, err := NewBot(config.Discord.Token, config, db, emailService)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}
	defer bot.Close()
	LogDebug("Bot instance created")

	// Start Discord bot
	log.Println("Connecting to Discord...")
	if err := bot.Start(); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	// Initialize web server
	log.Println("Initializing web server...")
	webServer := NewWebServer(config, db, bot)
	
	// Start web server in goroutine
	log.Printf("Web server starting on port %d", config.Server.Port)
	LogDebug("Base URL: %s", config.Server.BaseURL)
	go func() {
		if err := webServer.Start(); err != nil {
			log.Fatalf("Error starting web server: %v", err)
		}
	}()

	log.Println("")
	log.Println("════════════════════════════════════════════")
	log.Println("🛡️  Heimdall is now running!")
	log.Println("════════════════════════════════════════════")
	log.Println("Press CTRL-C to exit")
	log.Println("")

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("")
	log.Println("Shutting down gracefully...")
	LogInfo("Heimdall stopped")
}