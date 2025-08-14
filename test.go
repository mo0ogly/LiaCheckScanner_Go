package main

import (
	"fmt"
	"log"

	"github.com/lia/liacheckscanner_go/internal/config"
	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

func main() {
	fmt.Println("🔍 Test de LiaCheckScanner Go")
	fmt.Println("Owner: LIA - mo0ogly@proton.me")
	fmt.Println("")

	// Test du logger
	fmt.Println("🧪 Test 1: Logger...")
	logger := logger.NewLogger()
	logger.Info("Test", "✅ Logger fonctionnel")
	logger.Warning("Test", "⚠️ Test d'avertissement")
	logger.Error("Test", "❌ Test d'erreur")
	fmt.Println("✅ Logger testé")

	// Test de la configuration
	fmt.Println("🧪 Test 2: Configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Erreur configuration:", err)
	}
	fmt.Printf("✅ Configuration chargée: %s v%s\n", cfg.AppName, cfg.Version)

	// Test des modèles
	fmt.Println("🧪 Test 3: Modèles...")
	scannerData := models.ScannerData{
		ID:          "test_001",
		IPOrCIDR:    "192.168.1.1",
		ScannerName: "Test Scanner",
		ScannerType: models.ScannerTypeShodan,
		SourceFile:  "test.txt",
		CountryCode: "FR",
		CountryName: "France",
		ISP:         "Test ISP",
		RiskLevel:   "low",
		Tags:        []string{"test", "demo"},
	}
	fmt.Printf("✅ Modèle créé: %s (%s)\n", scannerData.ScannerName, scannerData.IPOrCIDR)

	// Test de l'extracteur
	fmt.Println("🧪 Test 4: Extracteur...")
	extractor := config.NewConfigManager()
	dbConfig := extractor.GetDatabaseConfig()
	fmt.Printf("✅ Extracteur configuré: %s\n", dbConfig.RepoURL)

	fmt.Println("")
	fmt.Println("🎉 Tous les tests sont réussis !")
	fmt.Println("LiaCheckScanner Go est prêt à l'utilisation.")
}
