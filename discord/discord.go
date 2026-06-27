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
	"minecraft-status-bot/discord/ui"

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
		if b.maintenance {
			b.state.MaintenanceAt = time.Now()
		}
		if err := state.Save(b.statePath, b.state); err != nil {
			slog.Error("failed to save state", "error", err)
		}

		slog.Info("sending shutdown embed to Discord")
		embed := ui.ShutdownEmbed(b.cfg.ServerIP, b.cfg.ServerPort, b.maintenance, b.state)
		b.sendEmbed(embed)

		slog.Info("closing Discord session")
		b.session.Close()
	})
}

func (b *Bot) SetMaintenance(on bool) {
	b.maintenanceMu.Lock()
	defer b.maintenanceMu.Unlock()
	if on && !b.maintenance {
		b.state.MaintenanceAt = time.Now()
	}
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
	embed := ui.LoadingEmbed(b.cfg.ServerIP, b.cfg.ServerPort, b.cfg.UpdateInterval)
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
			b.state.LastOffline = time.Now()
		}
		b.lastStatus = "offline"

		if b.OnStatusUpdate != nil {
			b.OnStatusUpdate(StatusUpdate{
				Online:   false,
				ServerIP: b.cfg.ServerIP,
			})
		}

		if b.IsMaintenance() {
			embed := ui.MaintenanceOfflineEmbed(b.cfg.ServerIP, b.cfg.ServerPort, b.state, b.cfg.UpdateInterval)
			b.editOrCreate(embed)
		} else {
			embed := ui.OfflineEmbed(b.cfg.ServerIP, b.cfg.ServerPort, b.state, b.cfg.UpdateInterval)
			b.editOrCreate(embed)
		}
		return
	}

	if b.lastStatus != "online" {
		slog.Info("server is back online")
		b.state.LastOnline = time.Now()
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

	ctx2 := context.Background()
	orynData, orynErr := b.orynClient.FetchPlayers(ctx2)
	playerList := "No players online."
	playerCount := 0
	if orynErr == nil && orynData != nil && len(orynData.Players) > 0 {
		playerCount = len(orynData.Players)
		var lines []string
		for i, p := range orynData.Players {
			lines = append(lines, fmt.Sprintf("`%d.` **%s** `(%dms)`", i+1, p.Username, p.Ping))
		}
		playerList = strings.Join(lines, "\n")
		if len(playerList) > 1024 {
			playerList = playerList[:1020] + "..."
		}
	} else if len(response.Players.Sample) > 0 {
		playerCount = len(response.Players.Sample)
		var lines []string
		for i, p := range response.Players.Sample {
			lines = append(lines, fmt.Sprintf("`%d.` **%s**", i+1, p.Name))
		}
		playerList = strings.Join(lines, "\n")
		if len(playerList) > 1024 {
			playerList = playerList[:1020] + "..."
		}
	}

	motd := response.MOTD.Clean
	if motd == "" {
		motd = "No message"
	}

	statusData := &ui.StatusData{
		Version:       response.Version.Name,
		Protocol:      response.Version.Protocol,
		PlayersOnline: response.Players.Online,
		PlayersMax:    response.Players.Max,
		Latency:       response.RoundTripLatency,
		MOTD:          motd,
		PlayerCount:   playerCount,
		PlayerList:    playerList,
	}

	if b.IsMaintenance() {
		embed := ui.MaintenanceEmbed(b.cfg.ServerIP, b.cfg.ServerPort, b.state, b.cfg.UpdateInterval)
		b.editOrCreate(embed)
	} else {
		embed := ui.OnlineEmbed(b.cfg.ServerIP, b.cfg.ServerPort, statusData, b.state, b.cfg.UpdateInterval)
		b.editOrCreate(embed)
	}
}

func (b *Bot) sendEmbed(embed *discordgo.MessageEmbed) {
	if b.statusMessage != nil {
		_, err := b.session.ChannelMessageEditEmbed(b.cfg.ChannelID, b.statusMessage.ID, embed)
		if err != nil {
			slog.Warn("failed to edit status message, sending new one", "error", err)
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
