package discord

import (
	"fmt"
	"log"
	"strings"
	"time"

	"minecraft-status-bot/config"
	"minecraft-status-bot/mcstatus"
	"minecraft-status-bot/orynapi"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session       *discordgo.Session
	cfg           *config.Config
	statusMessage *discordgo.Message
	lastStatus    string
	orynClient    *orynapi.Client
}

func New(cfg *config.Config) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	bot := &Bot{
		session:    session,
		cfg:        cfg,
		orynClient: orynapi.NewClient(cfg.OrynApiUrl),
	}

	session.AddHandler(bot.onReady)

	return bot, nil
}

func (b *Bot) Start() error {
	return b.session.Open()
}

func (b *Bot) Stop() {
	b.session.Close()
}

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("✅ Logged in as %s!\n", s.State.User.Username)

	b.fetchOrCreateStatusMessage()
	b.updateServerStatus()

	go func() {
		ticker := time.NewTicker(time.Duration(b.cfg.UpdateInterval) * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			b.updateServerStatus()
		}
	}()
}

func (b *Bot) fetchOrCreateStatusMessage() {
	_, err := b.session.Channel(b.cfg.ChannelID)
	if err != nil {
		log.Printf("❌ Error fetching channel: %v\n", err)
		return
	}

	messages, err := b.session.ChannelMessages(b.cfg.ChannelID, 10, "", "", "")
	if err != nil {
		log.Printf("❌ Error fetching messages: %v\n", err)
		return
	}

	for _, msg := range messages {
		if msg.Author.ID == b.session.State.User.ID {
			log.Println("🔄 Existing status message found, updating it...")
			b.statusMessage = msg
			return
		}
	}

	log.Println("📤 No existing message found, sending a new one...")
	embed := b.generateLoadingEmbed()
	msg, err := b.session.ChannelMessageSendEmbed(b.cfg.ChannelID, embed)
	if err != nil {
		log.Printf("❌ Error sending message: %v\n", err)
		return
	}
	b.statusMessage = msg
}

func (b *Bot) updateServerStatus() {
	response, err := mcstatus.Status(b.cfg.ServerIP, b.cfg.ServerPort)
	if err != nil {
		log.Printf("❌ Error fetching Minecraft server status: %v\n", err)
		if b.lastStatus != "offline" {
			log.Println("❌ Server is offline, updating message...")
		}
		b.lastStatus = "offline"
		b.sendOfflineEmbed()
		return
	}

	if b.lastStatus != "online" {
		log.Println("✅ Server is back online, updating message...")
	}
	b.lastStatus = "online"

	embed := b.buildOnlineEmbed(response)
	b.editOrCreate(embed)
}

func (b *Bot) buildOnlineEmbed(status *mcstatus.StatusResponse) *discordgo.MessageEmbed {
	orynData, err := b.orynClient.FetchPlayers()
	orynPlayerList := "No players online."
	if err == nil && orynData != nil && len(orynData.Players) > 0 {
		var lines []string
		for i, p := range orynData.Players {
			lines = append(lines, fmt.Sprintf("%d. %s (%dms)", i+1, p.Username, p.Ping))
		}
		orynPlayerList = strings.Join(lines, "\n")
		if len(orynPlayerList) > 1024 {
			orynPlayerList = orynPlayerList[:1020] + "..."
		}
	}

	motd := status.MOTD.Clean
	if motd == "" {
		motd = "No message"
	}

	return &discordgo.MessageEmbed{
		Title:       "🟢 Minecraft Server Online",
		Description: fmt.Sprintf("🌍 **Server IP:** `%s`", b.cfg.ServerIP),
		Color:       0x00FF00,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://api.mcsrvstat.us/icon/%s", b.cfg.ServerIP),
		},
		Image: &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("https://mcapi.us/server/image?theme=dark&ip=%s:%d", b.cfg.ServerIP, b.cfg.ServerPort),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📝 Version",
				Value:  status.Version.Name,
				Inline: true,
			},
			{
				Name:   "👥 Players",
				Value:  fmt.Sprintf("%d/%d", status.Players.Online, status.Players.Max),
				Inline: true,
			},
			{
				Name:   "📊 Ping",
				Value:  fmt.Sprintf("%dms", status.RoundTripLatency),
				Inline: true,
			},
			{
				Name:   "🎮 Players",
				Value:  orynPlayerList,
				Inline: false,
			},
			{
				Name:   "📢 MOTD",
				Value:  motd,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Last updated",
			IconURL: "https://cdn-icons-png.flaticon.com/512/906/906361.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (b *Bot) sendOfflineEmbed() {
	embed := &discordgo.MessageEmbed{
		Title:       "🔴 Minecraft Server Offline",
		Description: fmt.Sprintf("🚫 The server `%s` is currently offline or unreachable.", b.cfg.ServerIP),
		Color:       0xFF0000,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	b.editOrCreate(embed)
}

func (b *Bot) editOrCreate(embed *discordgo.MessageEmbed) {
	if b.statusMessage == nil {
		log.Println("⚠️ Status message missing! Resending...")
		b.fetchOrCreateStatusMessage()
	}

	_, err := b.session.ChannelMessageEditEmbed(b.cfg.ChannelID, b.statusMessage.ID, embed)
	if err != nil {
		log.Println("⚠️ Status message might have been deleted, creating a new one...")
		b.fetchOrCreateStatusMessage()
		if b.statusMessage != nil {
			b.session.ChannelMessageEditEmbed(b.cfg.ChannelID, b.statusMessage.ID, embed)
		}
	}
}

func (b *Bot) generateLoadingEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⏳ Fetching Minecraft server status...",
		Description: "Please wait while we fetch the latest server details.",
		Color:       0xFFFF00,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
