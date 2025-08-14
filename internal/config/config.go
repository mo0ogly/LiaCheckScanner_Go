package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// ConfigManager gère la configuration de l'application
type ConfigManager struct {
	config     *models.AppConfig
	configPath string
}

// NewConfigManager crée un nouveau gestionnaire de configuration
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

// LoadConfig charge la configuration depuis le fichier
func LoadConfig() (*models.AppConfig, error) {
	cm := NewConfigManager()
	return cm.Load()
}

// Load charge la configuration
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
			UpdateInterval: 24, // heures
		},
	}

	// Vérifier si le fichier de configuration existe
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Créer le fichier avec la configuration par défaut
		if err := cm.Save(defaultConfig); err != nil {
			return nil, fmt.Errorf("erreur lors de la création du fichier de configuration: %v", err)
		}
		return defaultConfig, nil
	}

	// Lire le fichier de configuration
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier de configuration: %v", err)
	}

	// Désérialiser la configuration
	var config models.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing de la configuration: %v", err)
	}

	cm.config = &config
	return &config, nil
}

// Save sauvegarde la configuration
func (cm *ConfigManager) Save(config *models.AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("erreur lors de la sérialisation de la configuration: %v", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("erreur lors de l'écriture du fichier de configuration: %v", err)
	}

	cm.config = config
	return nil
}

// GetConfig retourne la configuration actuelle
func (cm *ConfigManager) GetConfig() *models.AppConfig {
	return cm.config
}

// UpdateDatabaseConfig met à jour la configuration de la base de données
func (cm *ConfigManager) UpdateDatabaseConfig(dbConfig models.DatabaseConfig) error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database = dbConfig
	return cm.Save(cm.config)
}

// GetDatabaseConfig retourne la configuration de la base de données
func (cm *ConfigManager) GetDatabaseConfig() models.DatabaseConfig {
	if cm.config == nil {
		return models.DatabaseConfig{}
	}
	return cm.config.Database
}

// SetAPIKey définit la clé API
func (cm *ConfigManager) SetAPIKey(apiKey string) error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database.APIKey = apiKey
	return cm.Save(cm.config)
}

// GetAPIKey retourne la clé API
func (cm *ConfigManager) GetAPIKey() string {
	if cm.config == nil {
		return ""
	}
	return cm.config.Database.APIKey
}

// IsAPIEnabled vérifie si l'API est activée
func (cm *ConfigManager) IsAPIEnabled() bool {
	if cm.config == nil {
		return false
	}
	return cm.config.Database.EnableAPI
}

// EnableAPI active l'API
func (cm *ConfigManager) EnableAPI() error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database.EnableAPI = true
	return cm.Save(cm.config)
}

// DisableAPI désactive l'API
func (cm *ConfigManager) DisableAPI() error {
	if cm.config == nil {
		return fmt.Errorf("configuration non chargée")
	}

	cm.config.Database.EnableAPI = false
	return cm.Save(cm.config)
}
