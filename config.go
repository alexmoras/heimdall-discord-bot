package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Discord struct {
		Token          string `yaml:"token"`
		GuildID        string `yaml:"guild_id"`
		AdminRole      string `yaml:"admin_role"`
		MembersRole    string `yaml:"members_role"`    // Base role assigned to all verified users
		WelcomeMessage string `yaml:"welcome_message"` // Message sent to users when they join
	} `yaml:"discord"`

	Email struct {
		SMTPHost     string `yaml:"smtp_host"`
		SMTPPort     int    `yaml:"smtp_port"`
		SMTPUsername string `yaml:"smtp_username"`
		SMTPPassword string `yaml:"smtp_password"`
		FromAddress  string `yaml:"from_address"`
		FromName     string `yaml:"from_name"`
	} `yaml:"email"`

	Server struct {
		Port     int    `yaml:"port"`
		BaseURL  string `yaml:"base_url"` // e.g., https://yourdomain.com
		LogLevel string `yaml:"log_level"` // ERROR, WARN, INFO, DEBUG (default: INFO)
	} `yaml:"server"`

	Features struct {
		EnableTeamSelection bool `yaml:"enable_team_selection"` // Enable team/role selection during verification
	} `yaml:"features"`

	ApprovedDomains []string          `yaml:"approved_domains"`
	Teams           map[string]string `yaml:"teams"` // team name -> role ID
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}