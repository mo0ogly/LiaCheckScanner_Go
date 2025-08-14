package main

import (
	"os"

	"github.com/lia/liacheckscanner_go/internal/config"
	"github.com/lia/liacheckscanner_go/internal/gui"
	"github.com/lia/liacheckscanner_go/internal/logger"
)

// Version de l'application
const (
	Version = "1.0.0"
	AppName = "LiaCheckScanner"
	Owner   = "LIA - mo0ogly@proton.me"
)

// createRequiredDirectories creates all necessary directories for the application
func createRequiredDirectories() error {
	// List of required directories
	dirs := []string{
		"logs",
		"results",
		"data",
		"config",
		"assets/icons",
		"build",
		"build/data",
		"internet-scanners",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Create required directories first
	if err := createRequiredDirectories(); err != nil {
		os.Stderr.WriteString("❌ Erreur lors de la création des répertoires: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialiser le logger
	logger := logger.NewLogger()
	logger.Info("Main", "🚀 Démarrage de "+AppName+" v"+Version)
	logger.Info("Main", "👨‍💻 Owner: "+Owner)
	logger.Info("Main", "📁 Répertoires créés avec succès")

	// Charger la configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Main", "❌ Erreur lors du chargement de la configuration: "+err.Error())
		os.Exit(1)
	}
	logger.Info("Main", "✅ Configuration chargée avec succès")

	// Créer et lancer l'interface graphique
	app := gui.NewApp(cfg, logger)
	app.Run()

	logger.Info("Main", "👋 "+AppName+" fermé avec succès")
}
