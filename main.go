package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"minecraft-status-bot/config"
	"minecraft-status-bot/console"
	"minecraft-status-bot/discord"
)

const (
	botName     = "Minecraft Status Bot"
	botVersion  = "v1.0.0"
	stateFile   = "state.json"
)

func main() {
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	cons := console.New(botName, botVersion)
	cons.PrintBanner()

	cons.PrintInfo("Loading configuration...")
	bot, err := discord.New(cfg, stateFile)
	if err != nil {
		cons.PrintError(fmt.Sprintf("Failed to create bot: %v", err))
		os.Exit(1)
	}

	bot.OnStatusUpdate = func(update discord.StatusUpdate) {
		cons.PrintServerUpdate(update.Online, update.ServerIP, update.Players, update.MaxPlayers, update.Latency)
	}

	cons.PrintInfo("Starting Discord bot...")
	if err := bot.Start(); err != nil {
		cons.PrintError(fmt.Sprintf("Failed to start bot: %v", err))
		os.Exit(1)
	}

	cons.PrintSuccess("Bot started successfully!")
	cons.PrintInfo(fmt.Sprintf("Server: %s:%d", cfg.ServerIP, cfg.ServerPort))
	cons.PrintInfo(fmt.Sprintf("Channel: %s", cfg.ChannelID))
	if bot.IsMaintenance() {
		cons.PrintWarning("Restored maintenance mode from previous session")
		cons.PrintMaintenance(true)
	} else {
		cons.PrintMaintenance(false)
	}
	fmt.Println()

	cons.RegisterCommand("/mt", func(args string) {
		parts := strings.Fields(args)
		if len(parts) == 0 {
			cons.PrintWarning("Usage: /mt on | /mt off")
			return
		}

		switch strings.ToLower(parts[0]) {
		case "on":
			bot.SetMaintenance(true)
			cons.PrintMaintenance(true)
			cons.PrintSuccess("Maintenance mode enabled!")
		case "off":
			bot.SetMaintenance(false)
			cons.PrintMaintenance(false)
			cons.PrintSuccess("Maintenance mode disabled!")
		default:
			cons.PrintWarning(fmt.Sprintf("Unknown argument: %s. Use /mt on or /mt off", parts[0]))
		}
	})

	cons.RegisterCommand("/status", func(args string) {
		cons.PrintStatus(bot.IsMaintenance(), cfg.ServerIP, -1, -1, bot.IsMaintenance())
	})

	cons.RegisterCommand("/help", func(args string) {
		cons.PrintBanner()
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cons.RegisterCommand("/quit", func(args string) {
		cons.PrintInfo("Shutting down bot...")
		stop()
	})

	go func() {
		<-ctx.Done()
		cons.PrintInfo("\nReceived shutdown signal, sending shutdown embed...")
		bot.Stop()
		cons.PrintSuccess("Bot shutdown complete.")
		os.Exit(0)
	}()

	cons.StartInputLoop(func(input string) {
		if strings.HasPrefix(input, "/") {
			cons.ProcessCommand(input)
		} else {
			cons.PrintWarning(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", input))
		}
	})
}
