package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// ConfigManager manages loading, saving, and accessing the application configuration.
type ConfigManager struct {
	config     *models.AppConfig
	configPath string
}

// NewConfigManager creates a new ConfigManager and ensures the config directory exists.
func NewConfigManager() *ConfigManager {
	configDir := "./config"
	configPath := filepath.Join(configDir, "config.json")

	// Créer le dossier config s'il n'existe pas
	if err := os.MkdirAll(configDir, 0755); err != nil {
		panic(fmt.Sprintf("Impossible de créer le dossier config: %v", err))
	}

	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig loads the application configuration from the default config file path.
func LoadConfig() (*models.AppConfig, error) {
	cm := NewConfigManager()
	return cm.Load()
}

// Load reads and parses the configuration file, creating it with defaults if it does not exist.
func (cm *ConfigManager) Load() (*models.AppConfig, error) {
	// Configuration par défaut
	defaultConfig := &models.AppConfig{
		AppName:    "LiaCheckScanner",
		Version:    "1.0.0",
		Owner:      "LIA - mo0ogly@proton.me",
		Theme:      "dark",
		Language:   "fr",
		LogLevel:   "INFO",
		MaxLogSize: 10, // MB
		LogBackups: 5,
		Database: models.DatabaseConfig{
			RepoURL:        "https://github.com/six2dez/reconftw",
			LocalPath:      "./data/repository",
			ResultsDir:     "./results",
			LogsDir:        "./logs",
			APIKey:         "",
			EnableAPI:      false,
			APIThrottle:    1.0,
			AutoUpdate:     false,
			UpdateInterval: 24,  // heures
			CacheTTLHours:  168, // 7 days
		},
	}

	// Vérifier si le fichier de configuration existe
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Créer le fichier avec la configuration par défaut
		if err := cm.Save(defaultConfig); err != nil {
			return nil, fmt.Errorf("creating default config file: %w", err)
		}
		return defaultConfig, nil
	}

	// Lire le fichier de configuration
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Désérialiser la configuration
	var config models.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	cm.config = &config
	return &config, nil
}

// Save serializes the given configuration to JSON and writes it to the config file.
func (cm *ConfigManager) Save(config *models.AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config to JSON: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	cm.config = config
	return nil
}

// GetConfig returns the currently loaded application configuration.
func (cm *ConfigManager) GetConfig() *models.AppConfig {
	return cm.config
}

// UpdateDatabaseConfig updates the database section of the configuration and persists the change.
func (cm *ConfigManager) UpdateDatabaseConfig(dbConfig models.DatabaseConfig) error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database = dbConfig
	return cm.Save(cm.config)
}

// GetDatabaseConfig returns the database configuration section.
func (cm *ConfigManager) GetDatabaseConfig() models.DatabaseConfig {
	if cm.config == nil {
		return models.DatabaseConfig{}
	}
	return cm.config.Database
}

// SetAPIKey sets the API key in the database configuration and saves the change.
func (cm *ConfigManager) SetAPIKey(apiKey string) error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database.APIKey = apiKey
	return cm.Save(cm.config)
}

// GetAPIKey returns the currently configured API key.
func (cm *ConfigManager) GetAPIKey() string {
	if cm.config == nil {
		return ""
	}
	return cm.config.Database.APIKey
}

// IsAPIEnabled reports whether the API is enabled in the current configuration.
func (cm *ConfigManager) IsAPIEnabled() bool {
	if cm.config == nil {
		return false
	}
	return cm.config.Database.EnableAPI
}

// EnableAPI enables the API in the configuration and saves the change.
func (cm *ConfigManager) EnableAPI() error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database.EnableAPI = true
	return cm.Save(cm.config)
}

// DisableAPI disables the API in the configuration and saves the change.
func (cm *ConfigManager) DisableAPI() error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database.EnableAPI = false
	return cm.Save(cm.config)
}

// Validate checks that the given AppConfig has valid values.
// It returns an error describing the first validation failure, or nil if valid.
func Validate(cfg *models.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if strings.TrimSpace(cfg.AppName) == "" {
		return fmt.Errorf("AppName must not be empty")
	}

	if strings.TrimSpace(cfg.Version) == "" {
		return fmt.Errorf("Version must not be empty")
	}

	switch strings.ToUpper(cfg.LogLevel) {
	case "DEBUG", "INFO", "WARNING", "ERROR":
		// valid
	default:
		return fmt.Errorf("LogLevel must be one of DEBUG, INFO, WARNING, ERROR; got %q", cfg.LogLevel)
	}

	if cfg.MaxLogSize <= 0 {
		return fmt.Errorf("MaxLogSize must be > 0; got %d", cfg.MaxLogSize)
	}

	if cfg.LogBackups < 0 {
		return fmt.Errorf("LogBackups must be >= 0; got %d", cfg.LogBackups)
	}

	repoURL := strings.TrimSpace(cfg.Database.RepoURL)
	if repoURL == "" || (!strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://")) {
		return fmt.Errorf("Database.RepoURL must be a valid URL starting with http:// or https://; got %q", cfg.Database.RepoURL)
	}

	if cfg.Database.APIThrottle < 0 {
		return fmt.Errorf("Database.APIThrottle must be >= 0; got %f", cfg.Database.APIThrottle)
	}

	return nil
}
