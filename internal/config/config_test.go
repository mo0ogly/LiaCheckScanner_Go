package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// newTestConfigManager creates a ConfigManager that stores its config
// in a temporary directory, so tests never touch the real ./config folder.
func newTestConfigManager(t *testing.T) *ConfigManager {
	t.Helper()
	dir := t.TempDir()
	return &ConfigManager{
		configPath: filepath.Join(dir, "config.json"),
	}
}

// ----- Load / Save round-trip -----

func TestLoad_MissingFile_CreatesDefaults(t *testing.T) {
	cm := newTestConfigManager(t)

	cfg, err := cm.Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	// The file should now exist on disk.
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		t.Fatal("Load() should have created the config file, but it does not exist")
	}

	// Verify key default values.
	assertString(t, "AppName", "LiaCheckScanner", cfg.AppName)
	assertString(t, "Version", "1.0.0", cfg.Version)
	assertString(t, "Theme", "dark", cfg.Theme)
	assertString(t, "Language", "fr", cfg.Language)
	assertString(t, "LogLevel", "INFO", cfg.LogLevel)
	assertInt(t, "MaxLogSize", 10, cfg.MaxLogSize)
	assertInt(t, "LogBackups", 5, cfg.LogBackups)
	assertString(t, "Database.RepoURL", "https://github.com/MDMCK10/internet-scanners", cfg.Database.RepoURL)
	assertString(t, "Database.LocalPath", "./data/internet-scanners", cfg.Database.LocalPath)
	assertString(t, "Database.ResultsDir", "./results", cfg.Database.ResultsDir)
	assertString(t, "Database.LogsDir", "./logs", cfg.Database.LogsDir)
	assertFloat(t, "Database.APIThrottle", 1.0, cfg.Database.APIThrottle)
	assertInt(t, "Database.Parallelism", 4, cfg.Database.Parallelism)
	assertInt(t, "Database.UpdateInterval", 24, cfg.Database.UpdateInterval)

	if cfg.Database.EnableAPI {
		t.Error("Database.EnableAPI should default to false")
	}
	if cfg.Database.AutoUpdate {
		t.Error("Database.AutoUpdate should default to false")
	}
}

func TestLoad_ExistingFile_ReturnsStoredValues(t *testing.T) {
	cm := newTestConfigManager(t)

	// Write a custom config to disk first.
	custom := &models.AppConfig{
		AppName:    "CustomApp",
		Version:    "2.5.0",
		Theme:      "light",
		Language:   "en",
		LogLevel:   "DEBUG",
		MaxLogSize: 5,
		LogBackups: 3,
		Database: models.DatabaseConfig{
			RepoURL:   "https://example.com/repo",
			LocalPath: "/tmp/repo",
			EnableAPI: true,
			APIKey:    "secret-key-123",
		},
	}
	data, err := json.MarshalIndent(custom, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := cm.Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	assertString(t, "AppName", "CustomApp", cfg.AppName)
	assertString(t, "Version", "2.5.0", cfg.Version)
	assertString(t, "Theme", "light", cfg.Theme)
	assertString(t, "Language", "en", cfg.Language)
	assertString(t, "LogLevel", "DEBUG", cfg.LogLevel)
	assertString(t, "Database.RepoURL", "https://example.com/repo", cfg.Database.RepoURL)
	assertString(t, "Database.APIKey", "secret-key-123", cfg.Database.APIKey)
	if !cfg.Database.EnableAPI {
		t.Error("Database.EnableAPI should be true")
	}
}

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	cm := newTestConfigManager(t)

	// Write garbage to the config file.
	if err := os.WriteFile(cm.configPath, []byte("{not valid json!}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := cm.Load()
	if err == nil {
		t.Fatal("Load() should return an error for invalid JSON")
	}
}

func TestSave_WritesValidJSON(t *testing.T) {
	cm := newTestConfigManager(t)

	cfg := &models.AppConfig{
		AppName: "SaveTest",
		Version: "1.0.0",
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}

	if err := cm.Save(cfg); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	// Read back the file and verify.
	raw, err := os.ReadFile(cm.configPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var read models.AppConfig
	if err := json.Unmarshal(raw, &read); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	assertString(t, "AppName", "SaveTest", read.AppName)
	assertString(t, "Database.RepoURL", "https://example.com", read.Database.RepoURL)
}

func TestSave_SetsInternalConfig(t *testing.T) {
	cm := newTestConfigManager(t)

	cfg := &models.AppConfig{AppName: "InternalCheck"}

	if err := cm.Save(cfg); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	if cm.config == nil {
		t.Fatal("Save() should set cm.config")
	}
	assertString(t, "cm.config.AppName", "InternalCheck", cm.config.AppName)
}

// ----- GetConfig -----

func TestGetConfig_NilBeforeLoad(t *testing.T) {
	cm := newTestConfigManager(t)

	if cm.GetConfig() != nil {
		t.Error("GetConfig() should return nil before Load() is called")
	}
}

func TestGetConfig_AfterLoad(t *testing.T) {
	cm := newTestConfigManager(t)

	_, err := cm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := cm.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig() should not return nil after Load()")
	}
}

// ----- GetDatabaseConfig / UpdateDatabaseConfig -----

func TestGetDatabaseConfig_NilConfig(t *testing.T) {
	cm := newTestConfigManager(t)

	dbCfg := cm.GetDatabaseConfig()
	if dbCfg.RepoURL != "" {
		t.Errorf("Expected empty RepoURL when config is nil, got %q", dbCfg.RepoURL)
	}
}

func TestUpdateDatabaseConfig_NilConfig_ReturnsError(t *testing.T) {
	cm := newTestConfigManager(t)

	err := cm.UpdateDatabaseConfig(models.DatabaseConfig{RepoURL: "https://new.example.com"})
	if err == nil {
		t.Fatal("UpdateDatabaseConfig should return error when config is nil")
	}
}

func TestUpdateDatabaseConfig_Success(t *testing.T) {
	cm := newTestConfigManager(t)

	if _, err := cm.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	newDB := models.DatabaseConfig{
		RepoURL:   "https://updated-repo.example.com",
		LocalPath: "/updated/path",
		EnableAPI: true,
	}

	if err := cm.UpdateDatabaseConfig(newDB); err != nil {
		t.Fatalf("UpdateDatabaseConfig: %v", err)
	}

	got := cm.GetDatabaseConfig()
	assertString(t, "RepoURL", "https://updated-repo.example.com", got.RepoURL)
	assertString(t, "LocalPath", "/updated/path", got.LocalPath)
	if !got.EnableAPI {
		t.Error("EnableAPI should be true after update")
	}

	// Verify the change was persisted to disk.
	cm2 := &ConfigManager{configPath: cm.configPath}
	cfg2, err := cm2.Load()
	if err != nil {
		t.Fatalf("Load (second): %v", err)
	}
	assertString(t, "persisted RepoURL", "https://updated-repo.example.com", cfg2.Database.RepoURL)
}

// ----- SetAPIKey / GetAPIKey -----

func TestSetAPIKey_NilConfig_ReturnsError(t *testing.T) {
	cm := newTestConfigManager(t)

	err := cm.SetAPIKey("some-key")
	if err == nil {
		t.Fatal("SetAPIKey should return error when config is nil")
	}
}

func TestGetAPIKey_NilConfig_ReturnsEmpty(t *testing.T) {
	cm := newTestConfigManager(t)

	key := cm.GetAPIKey()
	if key != "" {
		t.Errorf("GetAPIKey should return empty string when config is nil, got %q", key)
	}
}

func TestSetAPIKey_Success(t *testing.T) {
	cm := newTestConfigManager(t)
	if _, err := cm.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := cm.SetAPIKey("my-api-key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	assertString(t, "GetAPIKey", "my-api-key", cm.GetAPIKey())

	// Verify persistence.
	cm2 := &ConfigManager{configPath: cm.configPath}
	cfg2, err := cm2.Load()
	if err != nil {
		t.Fatalf("Load (second): %v", err)
	}
	assertString(t, "persisted APIKey", "my-api-key", cfg2.Database.APIKey)
}

// ----- IsAPIEnabled / EnableAPI / DisableAPI -----

func TestIsAPIEnabled_NilConfig(t *testing.T) {
	cm := newTestConfigManager(t)

	if cm.IsAPIEnabled() {
		t.Error("IsAPIEnabled should return false when config is nil")
	}
}

func TestEnableAPI_NilConfig_ReturnsError(t *testing.T) {
	cm := newTestConfigManager(t)

	err := cm.EnableAPI()
	if err == nil {
		t.Fatal("EnableAPI should return error when config is nil")
	}
}

func TestDisableAPI_NilConfig_ReturnsError(t *testing.T) {
	cm := newTestConfigManager(t)

	err := cm.DisableAPI()
	if err == nil {
		t.Fatal("DisableAPI should return error when config is nil")
	}
}

func TestEnableDisableAPI_RoundTrip(t *testing.T) {
	cm := newTestConfigManager(t)
	if _, err := cm.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Default should be disabled.
	if cm.IsAPIEnabled() {
		t.Fatal("API should be disabled by default")
	}

	// Enable.
	if err := cm.EnableAPI(); err != nil {
		t.Fatalf("EnableAPI: %v", err)
	}
	if !cm.IsAPIEnabled() {
		t.Error("API should be enabled after EnableAPI()")
	}

	// Disable.
	if err := cm.DisableAPI(); err != nil {
		t.Fatalf("DisableAPI: %v", err)
	}
	if cm.IsAPIEnabled() {
		t.Error("API should be disabled after DisableAPI()")
	}
}

// ----- Load / Save round-trip from defaults -----

func TestLoad_CreatedDefaults_AreReadableBack(t *testing.T) {
	cm := newTestConfigManager(t)

	// First Load creates defaults.
	cfg1, err := cm.Load()
	if err != nil {
		t.Fatalf("Load (first): %v", err)
	}

	// Second Load reads back the file.
	cm2 := &ConfigManager{configPath: cm.configPath}
	cfg2, err := cm2.Load()
	if err != nil {
		t.Fatalf("Load (second): %v", err)
	}

	// They should match.
	assertString(t, "AppName", cfg1.AppName, cfg2.AppName)
	assertString(t, "Version", cfg1.Version, cfg2.Version)
	assertString(t, "Theme", cfg1.Theme, cfg2.Theme)
	assertString(t, "Language", cfg1.Language, cfg2.Language)
	assertString(t, "Database.RepoURL", cfg1.Database.RepoURL, cfg2.Database.RepoURL)
}

// ----- Validate -----

func TestValidate_NilConfig(t *testing.T) {
	err := Validate(nil)
	if err == nil {
		t.Fatal("Validate(nil) should return an error")
	}
}

func TestValidate_ValidDefaults(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 5,
		Database: models.DatabaseConfig{
			RepoURL:     "https://example.com/repo",
			APIThrottle: 1.0,
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() should pass for valid config, got: %v", err)
	}
}

func TestValidate_EmptyAppName(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject empty AppName")
	}
	if !strings.Contains(err.Error(), "AppName") {
		t.Errorf("error should mention AppName, got: %v", err)
	}
}

func TestValidate_EmptyVersion(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "   ",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject whitespace-only Version")
	}
	if !strings.Contains(err.Error(), "Version") {
		t.Errorf("error should mention Version, got: %v", err)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "TRACE",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject invalid LogLevel")
	}
	if !strings.Contains(err.Error(), "LogLevel") {
		t.Errorf("error should mention LogLevel, got: %v", err)
	}
}

func TestValidate_ValidLogLevels(t *testing.T) {
	for _, level := range []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL", "debug", "info", "warning", "error", "critical"} {
		cfg := &models.AppConfig{
			AppName:    "TestApp",
			Version:    "1.0.0",
			LogLevel:   level,
			MaxLogSize: 10,
			LogBackups: 0,
			Database: models.DatabaseConfig{
				RepoURL: "https://example.com",
			},
		}
		if err := Validate(cfg); err != nil {
			t.Errorf("Validate() should accept LogLevel %q, got: %v", level, err)
		}
	}
}

func TestValidate_MaxLogSizeZero(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 0,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject MaxLogSize == 0")
	}
	if !strings.Contains(err.Error(), "MaxLogSize") {
		t.Errorf("error should mention MaxLogSize, got: %v", err)
	}
}

func TestValidate_MaxLogSizeNegative(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: -5,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject negative MaxLogSize")
	}
}

func TestValidate_LogBackupsNegative(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: -1,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject negative LogBackups")
	}
	if !strings.Contains(err.Error(), "LogBackups") {
		t.Errorf("error should mention LogBackups, got: %v", err)
	}
}

func TestValidate_LogBackupsZero_IsValid(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() should accept LogBackups == 0, got: %v", err)
	}
}

func TestValidate_EmptyRepoURL(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject empty RepoURL")
	}
	if !strings.Contains(err.Error(), "RepoURL") {
		t.Errorf("error should mention RepoURL, got: %v", err)
	}
}

func TestValidate_RepoURLNoScheme(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "example.com/repo",
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject RepoURL without http:// or https://")
	}
}

func TestValidate_RepoURLWithHTTP(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL: "http://example.com/repo",
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() should accept http:// RepoURL, got: %v", err)
	}
}

func TestValidate_NegativeAPIThrottle(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL:     "https://example.com",
			APIThrottle: -0.5,
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should reject negative APIThrottle")
	}
	if !strings.Contains(err.Error(), "APIThrottle") {
		t.Errorf("error should mention APIThrottle, got: %v", err)
	}
}

func TestValidate_ZeroAPIThrottle_IsValid(t *testing.T) {
	cfg := &models.AppConfig{
		AppName:    "TestApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 0,
		Database: models.DatabaseConfig{
			RepoURL:     "https://example.com",
			APIThrottle: 0,
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() should accept APIThrottle == 0, got: %v", err)
	}
}

func TestLoad_InvalidConfig_ReturnsValidationError(t *testing.T) {
	cm := newTestConfigManager(t)

	// Write a config that will fail validation (MaxLogSize == 0).
	bad := &models.AppConfig{
		AppName:    "BadApp",
		Version:    "1.0.0",
		LogLevel:   "INFO",
		MaxLogSize: 0,
		Database: models.DatabaseConfig{
			RepoURL: "https://example.com",
		},
	}
	data, err := json.MarshalIndent(bad, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = cm.Load()
	if err == nil {
		t.Fatal("Load() should return a validation error for invalid config")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("error should mention validation, got: %v", err)
	}
}

// ----- Benchmarks -----

// BenchmarkLoad benchmarks loading the configuration from disk.
func BenchmarkLoad(b *testing.B) {
	cm := newBenchConfigManager(b)

	// Seed a config file so Load() reads from disk instead of creating defaults.
	cfg := &models.AppConfig{
		AppName:    "BenchApp",
		Version:    "1.0.0",
		Theme:      "dark",
		Language:   "en",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 5,
		Database: models.DatabaseConfig{
			RepoURL:   "https://example.com/repo",
			LocalPath: "/tmp/bench",
			EnableAPI: true,
			APIKey:    "bench-key",
		},
	}
	if err := cm.Save(cfg); err != nil {
		b.Fatalf("Save (seed): %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cm.Load()
		if err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

// BenchmarkSave benchmarks saving the configuration to disk.
func BenchmarkSave(b *testing.B) {
	cm := newBenchConfigManager(b)
	cfg := &models.AppConfig{
		AppName:    "BenchApp",
		Version:    "1.0.0",
		Theme:      "dark",
		Language:   "en",
		LogLevel:   "INFO",
		MaxLogSize: 10,
		LogBackups: 5,
		Database: models.DatabaseConfig{
			RepoURL:   "https://example.com/repo",
			LocalPath: "/tmp/bench",
			EnableAPI: true,
			APIKey:    "bench-key",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cm.Save(cfg); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

// newBenchConfigManager creates a ConfigManager that stores its config in a
// temporary directory, so benchmarks never touch the real ./config folder.
func newBenchConfigManager(b *testing.B) *ConfigManager {
	b.Helper()
	dir := b.TempDir()
	return &ConfigManager{
		configPath: filepath.Join(dir, "config.json"),
	}
}

// ----- Helpers -----

func assertString(t *testing.T, field, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: want %q, got %q", field, want, got)
	}
}

func assertInt(t *testing.T, field string, want, got int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: want %d, got %d", field, want, got)
	}
}

func assertFloat(t *testing.T, field string, want, got float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s: want %f, got %f", field, want, got)
	}
}
