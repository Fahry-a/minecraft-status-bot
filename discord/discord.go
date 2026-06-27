package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"minecraft-status-bot/config"
	"minecraft-status-bot/mcstatus"
	"minecraft-status-bot/orynapi"
	"minecraft-status-bot/state"

	"github.com/bwmarrin/discordgo"
)

type StatusUpdate struct {
	Online     bool
	ServerIP   string
	Players    int
	MaxPlayers int
	Latency    int64
}

type Bot struct {
	session        *discordgo.Session
	cfg            *config.Config
	statusMessage  *discordgo.Message
	lastStatus     string
	orynClient     *orynapi.Client
	maintenanceMu  sync.RWMutex
	maintenance    bool
	OnStatusUpdate func(StatusUpdate)
	cancel         context.CancelFunc
	done           chan struct{}
	shutdownOnce   sync.Once
	state          *state.State
	statePath      string
}

func New(cfg *config.Config, statePath string) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	st := state.Load(statePath)

	bot := &Bot{
		session:    session,
		cfg:        cfg,
		orynClient: orynapi.NewClient(cfg.OrynApiUrl),
		done:       make(chan struct{}),
		state:      st,
		statePath:  statePath,
	}

	bot.maintenance = st.Maintenance

	session.AddHandler(bot.onReady)

	return bot, nil
}

func (b *Bot) Start() error {
	return b.session.Open()
}

func (b *Bot) Stop() {
	b.shutdownOnce.Do(func() {
		slog.Info("stopping ticker goroutine")
		if b.cancel != nil {
			b.cancel()
		}
		<-b.done

		slog.Info("saving state before shutdown")
		b.state.Maintenance = b.maintenance
		if err := state.Save(b.statePath, b.state); err != nil {
			slog.Error("failed to save state", "error", err)
		}

		slog.Info("sending shutdown embed to Discord")
		b.sendShutdownEmbed()

		slog.Info("closing Discord session")
		b.session.Close()
	})
}

func (b *Bot) SetMaintenance(on bool) {
	b.maintenanceMu.Lock()
	defer b.maintenanceMu.Unlock()
	b.maintenance = on
}

func (b *Bot) IsMaintenance() bool {
	b.maintenanceMu.RLock()
	defer b.maintenanceMu.RUnlock()
	return b.maintenance
}

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	slog.Info("logged in", "user", s.State.User.Username, "maintenance", b.maintenance)

	b.fetchOrCreateStatusMessage()
	b.updateServerStatus()

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	go func() {
		defer close(b.done)
		ticker := time.NewTicker(time.Duration(b.cfg.UpdateInterval) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.updateServerStatus()
			}
		}
	}()
}

func (b *Bot) fetchOrCreateStatusMessage() {
	_, err := b.session.Channel(b.cfg.ChannelID)
	if err != nil {
		slog.Error("failed to fetch channel", "error", err)
		return
	}

	messages, err := b.session.ChannelMessages(b.cfg.ChannelID, 10, "", "", "")
	if err != nil {
		slog.Error("failed to fetch messages", "error", err)
		return
	}

	for _, msg := range messages {
		if msg.Author.ID == b.session.State.User.ID {
			slog.Info("found existing status message, updating")
			b.statusMessage = msg
			return
		}
	}

	slog.Info("no existing message found, sending new one")
	embed := b.generateLoadingEmbed()
	msg, err := b.session.ChannelMessageSendEmbed(b.cfg.ChannelID, embed)
	if err != nil {
		slog.Error("failed to send message", "error", err)
		return
	}
	b.statusMessage = msg
}

func (b *Bot) updateServerStatus() {
	ctx := context.Background()
	response, err := mcstatus.Status(ctx, b.cfg.ServerIP, b.cfg.ServerPort)
	if err != nil {
		slog.Error("failed to fetch server status", "error", err)
		if b.lastStatus != "offline" {
			slog.Info("server is offline")
		}
		b.lastStatus = "offline"

		if b.OnStatusUpdate != nil {
			b.OnStatusUpdate(StatusUpdate{
				Online:   false,
				ServerIP: b.cfg.ServerIP,
			})
		}

		if b.IsMaintenance() {
			b.sendMaintenanceOfflineEmbed()
		} else {
			b.sendOfflineEmbed()
		}
		return
	}

	if b.lastStatus != "online" {
		slog.Info("server is back online")
	}
	b.lastStatus = "online"

	if b.OnStatusUpdate != nil {
		b.OnStatusUpdate(StatusUpdate{
			Online:     true,
			ServerIP:   b.cfg.ServerIP,
			Players:    response.Players.Online,
			MaxPlayers: response.Players.Max,
			Latency:    response.RoundTripLatency,
		})
	}

	if b.IsMaintenance() {
		b.sendMaintenanceEmbed()
	} else {
		embed := b.buildOnlineEmbed(response)
		b.editOrCreate(embed)
	}
}

func (b *Bot) buildOnlineEmbed(status *mcstatus.StatusResponse) *discordgo.MessageEmbed {
	ctx := context.Background()
	orynData, err := b.orynClient.FetchPlayers(ctx)
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
				Name:   "👥 Players Online",
				Value:  fmt.Sprintf("%d/%d", status.Players.Online, status.Players.Max),
				Inline: true,
			},
			{
				Name:   "📊 Ping",
				Value:  fmt.Sprintf("%dms", status.RoundTripLatency),
				Inline: true,
			},
			{
				Name:   "📋 Player List",
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

func (b *Bot) sendMaintenanceEmbed() {
	embed := &discordgo.MessageEmbed{
		Title:       "🔧 Server Under Maintenance",
		Description: fmt.Sprintf("The server `%s` is currently under maintenance.\nPlease check back later.", b.cfg.ServerIP),
		Color:       0xFFAA00,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  "Maintenance in progress",
				Inline: true,
			},
			{
				Name:   "🔄 Updates",
				Value:  fmt.Sprintf("Auto-updates every %ds", b.cfg.UpdateInterval/1000),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Maintenance Mode Active",
			IconURL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	b.editOrCreate(embed)
}

func (b *Bot) sendMaintenanceOfflineEmbed() {
	embed := &discordgo.MessageEmbed{
		Title:       "🔧🔴 Maintenance + Offline",
		Description: fmt.Sprintf("The server `%s` is under maintenance and currently offline.\nPlease check back later.", b.cfg.ServerIP),
		Color:       0xCC5500,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔧 Maintenance",
				Value:  "Active",
				Inline: true,
			},
			{
				Name:   "🔴 Server",
				Value:  "Offline",
				Inline: true,
			},
			{
				Name:   "📝 Note",
				Value:  "Bot will resume status updates when maintenance ends.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Maintenance Mode Active",
			IconURL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	b.editOrCreate(embed)
}

func (b *Bot) sendShutdownEmbed() {
	title := "🔴 Bot Offline"
	statusText := "Bot Offline"
	color := 0x808080

	if b.maintenance {
		title = "🔧🔴 Bot Offline (Maintenance)"
		statusText = "Bot Offline + Maintenance Active"
		color = 0xCC5500
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("The Minecraft Status Bot for `%s` has been shut down.\nServer status updates are paused.", b.cfg.ServerIP),
		Color:       color,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Status",
				Value:  statusText,
				Inline: true,
			},
			{
				Name:   "🕐 Shutdown At",
				Value:  time.Now().Format("2006-01-02 15:04:05"),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Bot will resume on next startup",
			IconURL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if b.statusMessage != nil {
		_, err := b.session.ChannelMessageEditEmbed(b.cfg.ChannelID, b.statusMessage.ID, embed)
		if err != nil {
			slog.Warn("failed to edit status message with shutdown embed, sending new one", "error", err)
			b.session.ChannelMessageSendEmbed(b.cfg.ChannelID, embed)
		}
	} else {
		b.session.ChannelMessageSendEmbed(b.cfg.ChannelID, embed)
	}
}

func (b *Bot) editOrCreate(embed *discordgo.MessageEmbed) {
	if b.statusMessage == nil {
		slog.Warn("status message missing, resending")
		b.fetchOrCreateStatusMessage()
	}

	_, err := b.session.ChannelMessageEditEmbed(b.cfg.ChannelID, b.statusMessage.ID, embed)
	if err != nil {
		slog.Warn("status message might have been deleted, creating new one")
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
