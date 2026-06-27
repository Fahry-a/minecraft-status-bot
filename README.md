# Minecraft Status Bot - Go

Discord bot that monitors Minecraft server status using raw MC protocol ping.

## Build

```bash
go build -o minecraft-status-bot .
```

## Run

```bash
./minecraft-status-bot
# or with custom config path
./minecraft-status-bot /path/to/config.json
```

## Config

Edit `config.json`:

```json
{
  "token": "YOUR_DISCORD_BOT_TOKEN",
  "serverIP": "mc.oryn.my.id",
  "serverPort": 25565,
  "channelID": "YOUR_CHANNEL_ID",
  "orynApiUrl": "http://api.mc.oryn.my.id:80",
  "updateInterval": 10000
}
```

| Field | Description |
|-------|-------------|
| `token` | Discord bot token |
| `serverIP` | Minecraft server hostname |
| `serverPort` | Minecraft server port (default: 25565) |
| `channelID` | Discord channel ID for status display |
| `orynApiUrl` | OrynApi base URL for player list |
| `updateInterval` | Update interval in milliseconds |
