package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// Logger représente le système de logging
type Logger struct {
	mu       sync.Mutex
	logFile  *os.File
	logLevel models.LogLevel
	entries  []models.LogEntry
	maxSize  int // MB
	backups  int
}

// NewLogger crée un nouveau logger
func NewLogger() *Logger {
	logger := &Logger{
		logLevel: models.LogLevelInfo,
		maxSize:  10, // MB
		backups:  5,
	}

	// Créer le dossier logs s'il n'existe pas
	logsDir := "./logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("Erreur lors de la création du dossier logs: %v", err)
		return logger
	}

	// Ouvrir le fichier de log
	logPath := filepath.Join(logsDir, fmt.Sprintf("liacheckscanner_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Erreur lors de l'ouverture du fichier de log: %v", err)
		return logger
	}

	logger.logFile = file

	// Configurer le log standard pour écrire dans le fichier
	log.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return logger
}

// SetLogLevel définit le niveau de log
func (l *Logger) SetLogLevel(level models.LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

// GetLogLevel retourne le niveau de log actuel
func (l *Logger) GetLogLevel() models.LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logLevel
}

// shouldLog vérifie si le message doit être loggé selon le niveau
func (l *Logger) shouldLog(level models.LogLevel) bool {
	levels := map[models.LogLevel]int{
		models.LogLevelDebug:    0,
		models.LogLevelInfo:     1,
		models.LogLevelWarning:  2,
		models.LogLevelError:    3,
		models.LogLevelCritical: 4,
	}

	return levels[level] >= levels[l.logLevel]
}

// log enregistre un message de log
func (l *Logger) log(level models.LogLevel, component, message string, data map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
		Data:      data,
	}

	// Ajouter à la liste des entrées
	l.entries = append(l.entries, entry)

	// Limiter la taille de la liste
	if len(l.entries) > 1000 {
		l.entries = l.entries[500:]
	}

	// Formater le message pour la console
	levelEmoji := map[models.LogLevel]string{
		models.LogLevelDebug:    "🐛",
		models.LogLevelInfo:     "ℹ️",
		models.LogLevelWarning:  "⚠️",
		models.LogLevelError:    "❌",
		models.LogLevelCritical: "🚨",
	}

	emoji := levelEmoji[level]
	if emoji == "" {
		emoji = "📝"
	}

	// Afficher dans la console
	fmt.Printf("%s [%s] %s: %s\n", emoji, level, component, message)

	// Écrire dans le fichier JSON
	if l.logFile != nil {
		jsonData, err := json.Marshal(entry)
		if err == nil {
			l.logFile.Write(append(jsonData, '\n'))
		}
	}

	// Vérifier la taille du fichier et faire la rotation si nécessaire
	l.checkRotation()
}

// Debug enregistre un message de debug
func (l *Logger) Debug(component, message string, data ...map[string]interface{}) {
	var dataMap map[string]interface{}
	if len(data) > 0 {
		dataMap = data[0]
	}
	l.log(models.LogLevelDebug, component, message, dataMap)
}

// Info enregistre un message d'information
func (l *Logger) Info(component, message string, data ...map[string]interface{}) {
	var dataMap map[string]interface{}
	if len(data) > 0 {
		dataMap = data[0]
	}
	l.log(models.LogLevelInfo, component, message, dataMap)
}

// Warning enregistre un avertissement
func (l *Logger) Warning(component, message string, data ...map[string]interface{}) {
	var dataMap map[string]interface{}
	if len(data) > 0 {
		dataMap = data[0]
	}
	l.log(models.LogLevelWarning, component, message, dataMap)
}

// Error enregistre une erreur
func (l *Logger) Error(component, message string, data ...map[string]interface{}) {
	var dataMap map[string]interface{}
	if len(data) > 0 {
		dataMap = data[0]
	}
	l.log(models.LogLevelError, component, message, dataMap)
}

// Critical enregistre une erreur critique
func (l *Logger) Critical(component, message string, data ...map[string]interface{}) {
	var dataMap map[string]interface{}
	if len(data) > 0 {
		dataMap = data[0]
	}
	l.log(models.LogLevelCritical, component, message, dataMap)
}

// GetEntries retourne les entrées de log
func (l *Logger) GetEntries() []models.LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Retourner une copie pour éviter les problèmes de concurrence
	entries := make([]models.LogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// GetRecentEntries retourne les entrées récentes
func (l *Logger) GetRecentEntries(count int) []models.LogEntry {
	entries := l.GetEntries()
	if count > len(entries) {
		count = len(entries)
	}
	return entries[len(entries)-count:]
}

// ClearEntries efface les entrées de log
func (l *Logger) ClearEntries() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = nil
}

// checkRotation vérifie et effectue la rotation des logs
func (l *Logger) checkRotation() {
	if l.logFile == nil {
		return
	}

	// Obtenir les informations du fichier
	info, err := l.logFile.Stat()
	if err != nil {
		return
	}

	// Vérifier si la taille dépasse la limite (en bytes)
	maxSizeBytes := int64(l.maxSize * 1024 * 1024)
	if info.Size() < maxSizeBytes {
		return
	}

	// Fermer le fichier actuel
	l.logFile.Close()

	// Créer un nouveau fichier avec timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logsDir := filepath.Dir(l.logFile.Name())
	newLogPath := filepath.Join(logsDir, fmt.Sprintf("liacheckscanner_%s.log", timestamp))

	// Ouvrir le nouveau fichier
	file, err := os.OpenFile(newLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}

	l.logFile = file

	// Nettoyer les anciens fichiers de log
	l.cleanupOldLogs(logsDir)
}

// cleanupOldLogs nettoie les anciens fichiers de log
func (l *Logger) cleanupOldLogs(logsDir string) {
	// Cette fonction pourrait être implémentée pour supprimer les anciens fichiers
	// selon le nombre de backups configuré
}

// Close ferme le logger
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
