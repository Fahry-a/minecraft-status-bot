package main

import (
	"fmt"
	"log"
	"os"

	"minecraft-status-bot/config"
	"minecraft-status-bot/discord"
)

func main() {
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v\n", err)
	}

	fmt.Println("🚀 Starting Minecraft Status Bot...")

	bot, err := discord.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v\n", err)
	}

	if err := bot.Start(); err != nil {
		log.Fatalf("❌ Failed to start bot: %v\n", err)
	}
	defer bot.Stop()

	// Block forever
	select {}
}
