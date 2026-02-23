package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatTimestamp(ts string) string {
	if ts == "" {
		return "N/A"
	}
	// Jira timestamps: 2024-01-15T10:30:00.000+0000
	t, err := time.Parse("2006-01-02T15:04:05.000-0700", ts)
	if err != nil {
		// Try alternate format
		t, err = time.Parse("2006-01-02T15:04:05.000Z0700", ts)
		if err != nil {
			return ts
		}
	}
	return t.Format("2006-01-02 15:04")
}

func makeHyperlink(url, text string) string {
	// OSC 8 terminal hyperlink
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

func issueURL(baseURL, issueKey string) string {
	return strings.TrimRight(baseURL, "/") + "/browse/" + issueKey
}
