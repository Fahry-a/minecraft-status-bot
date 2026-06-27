package ui

import (
	"fmt"
	"strings"
	"time"

	"minecraft-status-bot/state"

	"github.com/bwmarrin/discordgo"
)

const separator = "━━━━━━━━━━━━━━━━"

func playerBar(online, max int) string {
	if max == 0 {
		return "░░░░░░░░░░"
	}
	percent := float64(online) / float64(max)
	filled := int(percent * 10)
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}

func durationSince(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds ago", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh %dm ago", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd %dh ago", int(d.Hours()/24), int(d.Hours())%24)
	}
}

func OnlineEmbed(serverIP string, serverPort int, status *StatusData, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	serverAddr := fmt.Sprintf("%s:%d", serverIP, serverPort)
	playersLine := fmt.Sprintf("👥 %d/%d Players", status.PlayersOnline, status.PlayersMax)
	pingLine := fmt.Sprintf("📶 %dms", status.Latency)
	versionLine := fmt.Sprintf("🎮 %s", status.Version)

	capacityPercent := 0
	if status.PlayersMax > 0 {
		capacityPercent = status.PlayersOnline * 100 / status.PlayersMax
	}
	bar := playerBar(status.PlayersOnline, status.PlayersMax)
	capacityLine := fmt.Sprintf("%s %d%%", bar, capacityPercent)

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("**%s**\n", serverAddr))
	desc.WriteString("\n")
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString(fmt.Sprintf("%s\n", playersLine))
	desc.WriteString(fmt.Sprintf("%s\n", pingLine))
	desc.WriteString(fmt.Sprintf("%s\n", versionLine))
	desc.WriteString("\n")
	desc.WriteString(fmt.Sprintf("```%s```", capacityLine))
	desc.WriteString("\n")
	desc.WriteString(separator)

	if status.MOTD != "" {
		desc.WriteString("\n")
		desc.WriteString(fmt.Sprintf("📢 %s", status.MOTD))
		desc.WriteString("\n")
		desc.WriteString(separator)
	}

	if status.PlayerCount > 0 {
		desc.WriteString("\n")
		desc.WriteString(fmt.Sprintf("**Online:**\n%s", status.PlayerList))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🟢 Minecraft Server Online",
		Description: desc.String(),
		Color:       0x00FF00,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://api.mcsrvstat.us/icon/%s", serverIP),
		},
		Image: &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("https://mcapi.us/server/image?theme=dark&ip=%s:%d", serverIP, serverPort),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Auto-updates every %ds", updateInterval/1000),
			IconURL: "https://cdn-icons-png.flaticon.com/512/906/906361.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return embed
}

func OfflineEmbed(serverIP string, serverPort int, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	serverAddr := fmt.Sprintf("%s:%d", serverIP, serverPort)

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("**%s**\n", serverAddr))
	desc.WriteString("\n")
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString("🔴 Server is currently offline or unreachable.\n")
	desc.WriteString(separator)

	if !st.LastOnline.IsZero() {
		desc.WriteString("\n")
		desc.WriteString(fmt.Sprintf("Last online: %s\n", durationSince(st.LastOnline)))
		desc.WriteString(fmt.Sprintf("Offline since: %s", durationSince(st.LastOffline)))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔴 Minecraft Server Offline",
		Description: desc.String(),
		Color:       0xFF0000,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Auto-updates every %ds", updateInterval/1000),
			IconURL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return embed
}

func MaintenanceEmbed(serverIP string, serverPort int, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	serverAddr := fmt.Sprintf("%s:%d", serverIP, serverPort)

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("**%s**\n", serverAddr))
	desc.WriteString("\n")
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString("🔧 Server is under maintenance.\n")
	desc.WriteString("Status updates are paused.\n")
	desc.WriteString(separator)

	if !st.MaintenanceAt.IsZero() {
		desc.WriteString("\n")
		desc.WriteString(fmt.Sprintf("Maintenance since: %s", durationSince(st.MaintenanceAt)))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔧 Server Under Maintenance",
		Description: desc.String(),
		Color:       0xFFAA00,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Auto-updates every %ds", updateInterval/1000),
			IconURL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return embed
}

func MaintenanceOfflineEmbed(serverIP string, serverPort int, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	serverAddr := fmt.Sprintf("%s:%d", serverIP, serverPort)

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("**%s**\n", serverAddr))
	desc.WriteString("\n")
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString("🔧 Maintenance active + 🔴 Server offline.\n")
	desc.WriteString("Please check back later.\n")
	desc.WriteString(separator)

	if !st.MaintenanceAt.IsZero() {
		desc.WriteString("\n")
		desc.WriteString(fmt.Sprintf("Maintenance since: %s", durationSince(st.MaintenanceAt)))
	}
	if !st.LastOffline.IsZero() {
		desc.WriteString("\n")
		desc.WriteString(fmt.Sprintf("Offline since: %s", durationSince(st.LastOffline)))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔧🔴 Maintenance + Server Offline",
		Description: desc.String(),
		Color:       0xCC5500,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Auto-updates every %ds", updateInterval/1000),
			IconURL: "https://cdn-icons-png.flaticon.com/512/2885/2885417.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return embed
}

func ShutdownEmbed(serverIP string, serverPort int, maintenance bool, st *state.State) *discordgo.MessageEmbed {
	serverAddr := fmt.Sprintf("%s:%d", serverIP, serverPort)

	title := "🔴 Bot Offline"
	color := 0x808080
	statusLine := "Bot is offline. Status updates are paused."

	if maintenance {
		title = "🔧🔴 Bot Offline (Maintenance)"
		color = 0xCC5500
		statusLine = "Bot is offline. Maintenance mode was active during shutdown."
	}

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("**%s**\n", serverAddr))
	desc.WriteString("\n")
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString(fmt.Sprintf("%s\n", statusLine))
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString(fmt.Sprintf("Shutdown at: **%s**\n", time.Now().Format("2006-01-02 15:04:05")))
	desc.WriteString(fmt.Sprintf("Bot will automatically restore previous state on next startup."))

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc.String(),
		Color:       color,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Bot will resume on next startup",
			IconURL: "https://cdn-icons-png.flaticon.com/512/1828/1828843.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return embed
}

func LoadingEmbed(serverIP string, serverPort, updateInterval int) *discordgo.MessageEmbed {
	serverAddr := fmt.Sprintf("%s:%d", serverIP, serverPort)

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("**%s**\n", serverAddr))
	desc.WriteString("\n")
	desc.WriteString(separator)
	desc.WriteString("\n")
	desc.WriteString("Fetching server status...\n")
	desc.WriteString(separator)

	return &discordgo.MessageEmbed{
		Title:       "⏳ Loading",
		Description: desc.String(),
		Color:       0xFFFF00,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Auto-updates every %ds", updateInterval/1000),
			IconURL: "https://cdn-icons-png.flaticon.com/512/906/906361.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

type StatusData struct {
	Version       string
	Protocol      int
	PlayersOnline int
	PlayersMax    int
	Latency       int64
	MOTD          string
	PlayerCount   int
	PlayerList    string
}
