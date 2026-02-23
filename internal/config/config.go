package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BaseURL  string
	Email    string
	APIToken string
}

// ConfigLocations returns the list of config file locations that are checked
// in order of priority (first found wins).
func ConfigLocations() []string {
	locations := []string{
		".env", // Current directory
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		locations = append(locations, filepath.Join(homeDir, ".config", "jira", ".env"))
	}

	return locations
}

// Load loads configuration from environment variables and optional .env files.
// The configFile parameter allows specifying a custom config file path.
// If empty, the default locations are checked in order:
//  1. .env in current directory
//  2. ~/.config/jira/.env
//
// Environment variables always take precedence over file values.
func Load(configFile string) (*Config, error) {
	if configFile != "" {
		if err := godotenv.Load(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
	} else {
		for _, loc := range ConfigLocations() {
			if _, err := os.Stat(loc); err == nil {
				_ = godotenv.Load(loc)
				break
			}
		}
	}

	baseURL := os.Getenv("JIRA_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("JIRA_BASE_URL not set.\n\n%s", configHelp())
	}
	baseURL = strings.TrimRight(baseURL, "/")

	email := os.Getenv("JIRA_EMAIL")
	if email == "" {
		return nil, fmt.Errorf("JIRA_EMAIL not set.\n\n%s", configHelp())
	}

	apiToken := os.Getenv("JIRA_API_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("JIRA_API_TOKEN not set.\n\n%s", configHelp())
	}

	return &Config{
		BaseURL:  baseURL,
		Email:    email,
		APIToken: apiToken,
	}, nil
}

func configHelp() string {
	locations := ConfigLocations()
	var sb strings.Builder

	sb.WriteString("Configuration can be provided via:\n")
	sb.WriteString("  1. Environment variables (JIRA_BASE_URL, JIRA_EMAIL, JIRA_API_TOKEN)\n")
	sb.WriteString("  2. A .env file in one of these locations:\n")
	for _, loc := range locations {
		sb.WriteString(fmt.Sprintf("     - %s\n", loc))
	}
	sb.WriteString("  3. A custom config file via --config flag\n")
	sb.WriteString("\nExample .env file:\n")
	sb.WriteString("  JIRA_BASE_URL=https://yourcompany.atlassian.net\n")
	sb.WriteString("  JIRA_EMAIL=you@example.com\n")
	sb.WriteString("  JIRA_API_TOKEN=your_api_token\n")
	sb.WriteString("\nGet your API token at: https://id.atlassian.com/manage-profile/security/api-tokens")

	return sb.String()
}

// PrintConfigHelp prints the configuration help message.
func PrintConfigHelp() {
	fmt.Println("Jira CLI Configuration")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println(configHelp())
}
