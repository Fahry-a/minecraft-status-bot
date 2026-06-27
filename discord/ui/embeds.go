package ui

import (
	"fmt"
	"strings"
	"time"

	"minecraft-status-bot/state"

	"github.com/bwmarrin/discordgo"
)

const (
	brandIconURL       = "https://cdn-icons-png.flaticon.com/512/906/906361.png"
	offlineIconURL     = "https://cdn-icons-png.flaticon.com/512/1828/1828843.png"
	maintenanceIconURL = "https://cdn-icons-png.flaticon.com/512/2885/2885417.png"
)

func serverAddress(serverIP string, serverPort int) string {
	return fmt.Sprintf("%s:%d", serverIP, serverPort)
}

func updateCadence(updateInterval int) string {
	if updateInterval <= 0 {
		return "Auto refresh enabled"
	}
	return fmt.Sprintf("Auto refresh • every %ds", updateInterval/1000)
}

func playerBar(online, max int) string {
	if max <= 0 {
		return "▱▱▱▱▱▱▱▱▱▱"
	}

	percent := float64(online) / float64(max)
	filled := int(percent*10 + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", 10-filled)
}

func capacityPercent(online, max int) int {
	if max <= 0 {
		return 0
	}
	return online * 100 / max
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

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("2006-01-02 15:04:05 MST")
}

func baseEmbed(title, description string, color int, iconURL, updateText string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    updateText,
			IconURL: iconURL,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func OnlineEmbed(serverIP string, serverPort int, status *StatusData, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	addr := serverAddress(serverIP, serverPort)
	capacity := capacityPercent(status.PlayersOnline, status.PlayersMax)
	activity := "Freshly detected"
	if st != nil && !st.LastOnline.IsZero() {
		activity = durationSince(st.LastOnline)
	}

	description := fmt.Sprintf(
		"### 🟢 Server Online\n`%s` is reachable and accepting players.\n\n**%s** `%d%%` capacity",
		addr,
		playerBar(status.PlayersOnline, status.PlayersMax),
		capacity,
	)

	embed := baseEmbed("Minecraft Status Dashboard", description, 0x2ECC71, brandIconURL, updateCadence(updateInterval))
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: fmt.Sprintf("https://api.mcsrvstat.us/icon/%s", serverIP)}
	embed.Image = &discordgo.MessageEmbedImage{URL: fmt.Sprintf("https://mcapi.us/server/image?theme=dark&ip=%s:%d", serverIP, serverPort)}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "👥 Players", Value: fmt.Sprintf("**%d/%d** online\n%d%% slot usage", status.PlayersOnline, status.PlayersMax, capacity), Inline: true},
		{Name: "📶 Latency", Value: fmt.Sprintf("**%dms**\nRound-trip ping", status.Latency), Inline: true},
		{Name: "🎮 Version", Value: fmt.Sprintf("**%s**\nProtocol `%d`", status.Version, status.Protocol), Inline: true},
		{Name: "📌 Address", Value: fmt.Sprintf("`%s`", addr), Inline: true},
		{Name: "🕒 Online Since", Value: activity, Inline: true},
		{Name: "💬 MOTD", Value: status.MOTD, Inline: false},
	}

	if status.PlayerCount > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: fmt.Sprintf("🧑 Active Players (%d)", status.PlayerCount), Value: status.PlayerList, Inline: false})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "🧑 Active Players", Value: "No players are currently online.", Inline: false})
	}

	return embed
}

func OfflineEmbed(serverIP string, serverPort int, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	addr := serverAddress(serverIP, serverPort)
	embed := baseEmbed(
		"Minecraft Status Dashboard",
		fmt.Sprintf("### 🔴 Server Offline\n`%s` cannot be reached right now. The bot will keep checking and update this panel automatically.", addr),
		0xE74C3C,
		offlineIconURL,
		updateCadence(updateInterval),
	)
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: offlineIconURL}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "📌 Address", Value: fmt.Sprintf("`%s`", addr), Inline: true},
		{Name: "🧭 Status", Value: "Offline / unreachable", Inline: true},
		{Name: "🛠️ Next Action", Value: "Check server host, network, whitelist, or firewall rules.", Inline: false},
	}
	if st != nil && !st.LastOnline.IsZero() {
		embed.Fields = append(embed.Fields,
			&discordgo.MessageEmbedField{Name: "✅ Last Seen Online", Value: durationSince(st.LastOnline), Inline: true},
			&discordgo.MessageEmbedField{Name: "⛔ Offline Since", Value: durationSince(st.LastOffline), Inline: true},
		)
	}
	return embed
}

func MaintenanceEmbed(serverIP string, serverPort int, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	addr := serverAddress(serverIP, serverPort)
	embed := baseEmbed("Minecraft Status Dashboard", fmt.Sprintf("### 🔧 Maintenance Mode\n`%s` is online, but public status is currently paused for maintenance.", addr), 0xF1C40F, maintenanceIconURL, updateCadence(updateInterval))
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: maintenanceIconURL}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "📌 Address", Value: fmt.Sprintf("`%s`", addr), Inline: true},
		{Name: "🧭 Status", Value: "Maintenance active", Inline: true},
		{Name: "ℹ️ Notice", Value: "Updates are intentionally muted until maintenance mode is disabled.", Inline: false},
	}
	if st != nil && !st.MaintenanceAt.IsZero() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "🕒 Maintenance Since", Value: durationSince(st.MaintenanceAt), Inline: true})
	}
	return embed
}

func MaintenanceOfflineEmbed(serverIP string, serverPort int, st *state.State, updateInterval int) *discordgo.MessageEmbed {
	addr := serverAddress(serverIP, serverPort)
	embed := baseEmbed("Minecraft Status Dashboard", fmt.Sprintf("### 🔧🔴 Maintenance + Offline\n`%s` is under maintenance and currently unreachable.", addr), 0xD35400, maintenanceIconURL, updateCadence(updateInterval))
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: maintenanceIconURL}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "📌 Address", Value: fmt.Sprintf("`%s`", addr), Inline: true},
		{Name: "🧭 Status", Value: "Maintenance + offline", Inline: true},
		{Name: "ℹ️ Notice", Value: "Please check back later while maintenance is completed.", Inline: false},
	}
	if st != nil && !st.MaintenanceAt.IsZero() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "🕒 Maintenance Since", Value: durationSince(st.MaintenanceAt), Inline: true})
	}
	if st != nil && !st.LastOffline.IsZero() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "⛔ Offline Since", Value: durationSince(st.LastOffline), Inline: true})
	}
	return embed
}

func ShutdownEmbed(serverIP string, serverPort int, maintenance bool, st *state.State) *discordgo.MessageEmbed {
	addr := serverAddress(serverIP, serverPort)
	color := 0x95A5A6
	icon := offlineIconURL
	stateText := "Bot offline"
	description := fmt.Sprintf("### ⚫ Bot Stopped\n`%s` will no longer receive live status updates until the bot starts again.", addr)
	if maintenance {
		color = 0xD35400
		icon = maintenanceIconURL
		stateText = "Bot offline while maintenance was active"
		description = fmt.Sprintf("### 🔧⚫ Bot Stopped During Maintenance\n`%s` will restore maintenance state on the next startup.", addr)
	}

	embed := baseEmbed("Minecraft Status Dashboard", description, color, icon, "Bot will resume on next startup")
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: icon}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "📌 Address", Value: fmt.Sprintf("`%s`", addr), Inline: true},
		{Name: "🧭 Status", Value: stateText, Inline: true},
		{Name: "🕒 Shutdown At", Value: formatTime(time.Now()), Inline: false},
	}
	if st != nil && !st.LastOnline.IsZero() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "✅ Last Online", Value: durationSince(st.LastOnline), Inline: true})
	}
	return embed
}

func LoadingEmbed(serverIP string, serverPort, updateInterval int) *discordgo.MessageEmbed {
	addr := serverAddress(serverIP, serverPort)
	embed := baseEmbed("Minecraft Status Dashboard", fmt.Sprintf("### ⏳ Loading Server Data\n`%s` is being checked. The dashboard will refresh shortly.", addr), 0x3498DB, brandIconURL, updateCadence(updateInterval))
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "📌 Address", Value: fmt.Sprintf("`%s`", addr), Inline: true},
		{Name: "🧭 Status", Value: "Fetching latest ping result", Inline: true},
	}
	return embed
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
