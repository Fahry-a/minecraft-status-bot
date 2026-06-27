package console

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorGray    = "\033[90m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
)

type CommandHandler func(args string)

type Console struct {
	mu       sync.Mutex
	scanner  *bufio.Scanner
	commands map[string]CommandHandler
	botName  string
	version  string
}

func New(botName, version string) *Console {
	return &Console{
		scanner:  bufio.NewScanner(os.Stdin),
		commands: make(map[string]CommandHandler),
		botName:  botName,
		version:  version,
	}
}

func (c *Console) RegisterCommand(name string, handler CommandHandler) {
	c.commands[name] = handler
}

func (c *Console) PrintBanner() {
	fmt.Println()
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════════╗%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s║%s                                                              %s║%s\n", colorCyan, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s● %s%s%s%s%-44s%s   %s║%s\n", colorCyan, colorReset, colorGreen, colorBold, colorWhite, c.botName, colorReset, "", colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s  %s%s%-44s%s   %s║%s\n", colorCyan, colorReset, colorReset, colorDim, c.version, colorReset, "", colorCyan, colorReset)
	fmt.Printf("%s║%s                                                              %s║%s\n", colorCyan, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %sCommands:%s                                                %s║%s\n", colorCyan, colorReset, colorBold, colorYellow, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s/mt on%s    %s- Enable maintenance mode                  %s║%s\n", colorCyan, colorReset, colorCyan, colorGreen, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s/mt off%s   %s- Disable maintenance mode                 %s║%s\n", colorCyan, colorReset, colorCyan, colorGreen, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s/status%s  %s- Show bot status                           %s║%s\n", colorCyan, colorReset, colorCyan, colorGreen, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s/help%s    %s- Show this help menu                        %s║%s\n", colorCyan, colorReset, colorCyan, colorGreen, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s   %s/quit%s    %s- Shutdown bot                               %s║%s\n", colorCyan, colorReset, colorCyan, colorGreen, colorReset, colorCyan, colorReset)
	fmt.Printf("%s║%s                                                              %s║%s\n", colorCyan, colorReset, colorCyan, colorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════════╝%s\n", colorBold, colorCyan, colorReset)
	fmt.Println()
}

func (c *Console) PrintSuccess(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("  %s✓%s %s\n", colorGreen, colorReset, msg)
}

func (c *Console) PrintError(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, msg)
}

func (c *Console) PrintInfo(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("  %s●%s %s\n", colorBlue, colorReset, msg)
}

func (c *Console) PrintWarning(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Printf("  %s⚠%s %s\n", colorYellow, colorReset, msg)
}

func (c *Console) PrintMaintenance(mode bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if mode {
		fmt.Printf("  %s🔧 Maintenance Mode: %sON%s\n", colorYellow, colorBold+colorRed, colorReset)
	} else {
		fmt.Printf("  %s🔧 Maintenance Mode: %sOFF%s\n", colorYellow, colorBold+colorGreen, colorReset)
	}
}

func (c *Console) PrintStatus(online bool, serverIP string, players, mtMode bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	statusStr := fmt.Sprintf("%sOFFLINE%s", colorRed, colorReset)
	if online {
		statusStr = fmt.Sprintf("%sONLINE%s", colorGreen, colorReset)
	}

	mtStr := fmt.Sprintf("%sOFF%s", colorGreen, colorReset)
	if mtMode {
		mtStr = fmt.Sprintf("%sON%s", colorRed, colorReset)
	}

	playersStr := "Unknown"
	if players {
		playersStr = "Online"
	}

	fmt.Printf("\n  %s┌─────────────────────────────────────────┐%s\n", colorGray, colorReset)
	fmt.Printf("  %s│%s  %sBot Status%s                          %s│%s\n", colorGray, colorReset, colorBold, colorReset, colorGray, colorReset)
	fmt.Printf("  %s├─────────────────────────────────────────┤%s\n", colorGray, colorReset)
	fmt.Printf("  %s│%s  Server:    %s%-26s%s %s│%s\n", colorGray, colorReset, colorCyan, serverIP, colorReset, colorGray, colorReset)
	fmt.Printf("  %s│%s  Status:    %-26s %s│%s\n", colorGray, colorReset, statusStr, colorGray, colorReset)
	fmt.Printf("  %s│%s  Players:   %-26s %s│%s\n", colorGray, colorReset, playersStr, colorGray, colorReset)
	fmt.Printf("  %s│%s  Maint:     %-26s %s│%s\n", colorGray, colorReset, mtStr, colorGray, colorReset)
	fmt.Printf("  %s└─────────────────────────────────────────┘%s\n\n", colorGray, colorReset)
}

func (c *Console) PrintServerUpdate(online bool, serverIP string, playerCount, maxPlayers int, latency int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Format("15:04:05")
	if online {
		fmt.Printf("  %s[%s]%s %s● %s%sOnline%s | %s%d/%d players%s | %s%dms%s\n",
			colorGray, now, colorReset,
			colorGreen,
			colorDim, colorReset, colorReset,
			colorCyan, playerCount, maxPlayers, colorReset,
			colorYellow, latency, colorReset,
		)
	} else {
		fmt.Printf("  %s[%s]%s %s● %sOffline%s\n",
			colorGray, now, colorReset,
			colorRed,
			colorDim, colorReset,
		)
	}
}

func (c *Console) PrintPrompt() {
	fmt.Printf("  %s❯%s ", colorCyan, colorReset)
}

func (c *Console) StartInputLoop(handler func(input string)) {
	for {
		c.PrintPrompt()
		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		handler(input)
	}
}

func (c *Console) ProcessCommand(input string) bool {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	if handler, ok := c.commands[cmd]; ok {
		handler(args)
		return true
	}

	c.PrintWarning(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd))
	return false
}
