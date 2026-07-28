package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func prepareSessionBanner(now time.Time, cmd *cobra.Command) (greeting, lastUse string) {
	path := sessionStatePath()
	prev, err := loadSessionState(path)
	if err == nil {
		lastAt := prev.LastCommandAt
		if lastAt.IsZero() {
			lastAt = prev.LastLoginAt
		}
		if !lastAt.IsZero() {
			if strings.TrimSpace(prev.LastCommand) != "" {
				lastUse = cf("cli.session.last_use", lastAt.Local().Format("Mon Jan 02 15:04:05 2006"), prev.LastCommand)
			} else {
				lastUse = cf("cli.session.last_use_simple", lastAt.Local().Format("Mon Jan 02 15:04:05 2006"))
			}
		}
	}

	greeting = cf(statusGreetingKeyForHour(now.Hour()), resolveSystemUserName())
	_ = saveSessionState(path, cliSessionState{
		LastCommandAt: now,
		LastCommand:   cmd.CommandPath(),
		LastLoginAt:   now, // legacy compatibility
	})
	return greeting, lastUse
}

func statusGreetingKeyForHour(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "cli.status.greeting.morning"
	case hour >= 12 && hour < 20:
		return "cli.status.greeting.afternoon"
	default:
		return "cli.status.greeting.night"
	}
}

func printSessionBanner() {
	if sessionGreetingLine != "" {
		fmt.Println("  " + styleDim.Render(sessionGreetingLine))
	}
	if sessionLastUseLine != "" {
		fmt.Println("  " + styleDim.Render(sessionLastUseLine))
	}
	if sessionGreetingLine != "" || sessionLastUseLine != "" {
		fmt.Println()
	}
}
