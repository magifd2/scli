// Package config manages scli configuration and token resolution.
package config

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/magifd2/scli/internal/keychain"
)

const (
	configDirName = ".config/scli"
	configFile    = "config.json"
	dotEnvFile    = ".env"
)

// WorkspaceConfig holds per-workspace settings.
type WorkspaceConfig struct {
	// Token stores the token in plaintext. Leave empty to use the OS keychain.
	Token  string `json:"token,omitempty"`
	TeamID string `json:"team_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// Config is the top-level scli configuration structure.
type Config struct {
	DefaultWorkspace string                     `json:"default_workspace,omitempty"`
	Workspaces       map[string]WorkspaceConfig `json:"workspaces"`
}

// Manager reads and writes the scli config file.
type Manager struct {
	configPath string
}

// DefaultManager returns a Manager using the standard config path (~/.config/scli/config.json).
func DefaultManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}
	return &Manager{configPath: filepath.Join(home, configDirName, configFile)}, nil
}

// NewManager returns a Manager with a custom config path. Intended for tests.
func NewManager(configPath string) *Manager {
	return &Manager{configPath: configPath}
}

// ConfigDir returns the directory containing the config file.
func (m *Manager) ConfigDir() string {
	return filepath.Dir(m.configPath)
}

// Load reads the config file. Returns an empty Config if the file does not exist.
func (m *Manager) Load() (*Config, error) {
	data, err := os.ReadFile(m.configPath)
	if os.IsNotExist(err) {
		return &Config{Workspaces: make(map[string]WorkspaceConfig)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	if cfg.Workspaces == nil {
		cfg.Workspaces = make(map[string]WorkspaceConfig)
	}
	return &cfg, nil
}

// Save writes cfg to disk, creating parent directories as needed.
// The file is written with mode 0600 (owner read/write only).
func (m *Manager) Save(cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(m.configPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// ResolveToken returns the token for workspace using the following priority chain:
//  1. Environment variable SLACK_TOKEN_<WORKSPACE> (upper-cased, hyphens → underscores)
//  2. Environment variable SLACK_TOKEN
//  3. .env file in the current directory
//  4. .env file in the config directory (~/.config/scli/.env)
//  5. Token field in config.json for the workspace
//  6. OS keychain (via ks)
func (m *Manager) ResolveToken(workspace string, ks keychain.Store) (string, error) {
	envKey := "SLACK_TOKEN_" + strings.ToUpper(strings.ReplaceAll(workspace, "-", "_"))

	// 1 & 2: OS environment variables
	if token := os.Getenv(envKey); token != "" {
		return token, nil
	}
	if token := os.Getenv("SLACK_TOKEN"); token != "" {
		return token, nil
	}

	// 3 & 4: .env files (current dir takes precedence over config dir)
	dotEnvVars := readDotEnvFiles([]string{
		filepath.Join(m.ConfigDir(), dotEnvFile),
		dotEnvFile,
	})
	if token := dotEnvVars[envKey]; token != "" {
		return token, nil
	}
	if token := dotEnvVars["SLACK_TOKEN"]; token != "" {
		return token, nil
	}

	// 5: Config file
	cfg, err := m.Load()
	if err != nil {
		return "", err
	}
	if ws, ok := cfg.Workspaces[workspace]; ok && ws.Token != "" {
		return ws.Token, nil
	}

	// 6: OS keychain
	token, err := ks.Get(workspace)
	if err != nil {
		return "", fmt.Errorf("no token found for workspace %q — run: scli auth login --workspace %s", workspace, workspace)
	}
	return token, nil
}

// readDotEnvFiles reads the given .env files and merges them.
// Files listed later take precedence over earlier ones.
func readDotEnvFiles(paths []string) map[string]string {
	result := make(map[string]string)
	for _, p := range paths {
		vars, err := godotenv.Read(p)
		if err != nil {
			continue // file not found or unreadable — skip silently
		}
		maps.Copy(result, vars)
	}
	return result
}
