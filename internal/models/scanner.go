package models

import (
	"time"
)

// ScannerType représente le type de scanner
type ScannerType string

const (
	ScannerTypeUnknown      ScannerType = "unknown"
	ScannerTypeShodan       ScannerType = "shodan"
	ScannerTypeCensys       ScannerType = "censys"
	ScannerTypeBinaryEdge   ScannerType = "binaryedge"
	ScannerTypeRapid7       ScannerType = "rapid7"
	ScannerTypeShadowServer ScannerType = "shadowserver"
	ScannerTypeOther        ScannerType = "other"
)

// ScannerData représente les données d'un scanner
type ScannerData struct {
	ID                   string      `json:"id" csv:"ID"`
	IPOrCIDR             string      `json:"ip_or_cidr" csv:"IP/CIDR"`
	ScannerName          string      `json:"scanner_name" csv:"Scanner Name"`
	ScannerType          ScannerType `json:"scanner_type" csv:"Scanner Type"`
	SourceFile           string      `json:"source_file" csv:"Source File"`
	CountryCode          string      `json:"country_code" csv:"Country Code"`
	CountryName          string      `json:"country_name" csv:"Country Name"`
	ISP                  string      `json:"isp" csv:"ISP"`
	Organization         string      `json:"organization" csv:"Organization"`
	AbuseConfidenceScore int         `json:"abuse_confidence_score" csv:"Abuse Confidence Score"`
	AbuseReports         int         `json:"abuse_reports" csv:"Abuse Reports"`
	UsageType            string      `json:"usage_type" csv:"Usage Type"`
	Domain               string      `json:"domain" csv:"Domain"`
	// RDAP / WHOIS-like details
	RDAPName          string `json:"rdap_name" csv:"RDAP Name"`
	RDAPHandle        string `json:"rdap_handle" csv:"RDAP Handle"`
	RDAPCIDR          string `json:"rdap_cidr" csv:"RDAP CIDR"`
	Registry          string `json:"registry" csv:"RDAP Registry"`
	StartAddress      string `json:"start_address" csv:"Start Address"`
	EndAddress        string `json:"end_address" csv:"End Address"`
	IPVersion         string `json:"ip_version" csv:"IP Version"`
	RDAPType          string `json:"rdap_type" csv:"RDAP Type"`
	ParentHandle      string `json:"parent_handle" csv:"Parent Handle"`
	EventRegistration string `json:"event_registration" csv:"Event Registration"`
	EventLastChanged  string `json:"event_last_changed" csv:"Event Last Changed"`
	// ASN
	ASN    string `json:"asn" csv:"ASN"`
	ASName string `json:"as_name" csv:"AS Name"`
	// DNS reverse
	ReverseDNS string `json:"reverse_dns" csv:"Reverse DNS"`
	// Contacts
	AbuseEmail string    `json:"abuse_email" csv:"Abuse Email"`
	TechEmail  string    `json:"tech_email" csv:"Tech Email"`
	LastSeen   time.Time `json:"last_seen" csv:"Last Seen"`
	FirstSeen  time.Time `json:"first_seen" csv:"First Seen"`
	Tags       []string  `json:"tags" csv:"Tags"`
	Notes      string    `json:"notes" csv:"Notes"`
	RiskLevel  string    `json:"risk_level" csv:"Risk Level"`
	ExportDate time.Time `json:"export_date" csv:"Export Date"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RDAPCacheEntry permet de persister les résultats RDAP/Geo pour une IP
type RDAPCacheEntry struct {
	RDAPName          string `json:"rdap_name"`
	RDAPHandle        string `json:"rdap_handle"`
	RDAPCIDR          string `json:"rdap_cidr"`
	Registry          string `json:"registry"`
	StartAddress      string `json:"start_address"`
	EndAddress        string `json:"end_address"`
	IPVersion         string `json:"ip_version"`
	RDAPType          string `json:"rdap_type"`
	ParentHandle      string `json:"parent_handle"`
	EventRegistration string `json:"event_registration"`
	EventLastChanged  string `json:"event_last_changed"`
	ASN               string `json:"asn"`
	ASName            string `json:"as_name"`
	ReverseDNS        string `json:"reverse_dns"`
	CountryCode       string `json:"country_code"`
	CountryName       string `json:"country_name"`
	ISP               string `json:"isp"`
	Organization      string `json:"organization"`
	AbuseEmail        string `json:"abuse_email"`
	TechEmail         string `json:"tech_email"`
	CachedAt          string `json:"cached_at"`
}

// RDAPProgressTracker permet de reprendre les enrichissements RDAP là où on s'est arrêté
type RDAPProgressTracker struct {
	TotalRecords     int      `json:"total_records"`
	ProcessedRecords int      `json:"processed_records"`
	ProcessedIPs     []string `json:"processed_ips"`
	StartedAt        string   `json:"started_at"`
	LastUpdatedAt    string   `json:"last_updated_at"`
	Workers          int      `json:"workers"`
	Throttle         float64  `json:"throttle"`
	Completed        bool     `json:"completed"`
}

// DatabaseConfig représente la configuration de la base de données
type DatabaseConfig struct {
	RepoURL        string   `json:"repo_url"`
	LocalPath      string   `json:"local_path"`
	ResultsDir     string   `json:"results_dir"`
	LogsDir        string   `json:"logs_dir"`
	APIKey         string   `json:"api_key"`
	EnableAPI      bool     `json:"enable_api"`
	APIThrottle    float64  `json:"api_throttle"`
	Parallelism    int      `json:"parallelism"`
	Registries     []string `json:"registries"`
	AutoUpdate     bool     `json:"auto_update"`
	UpdateInterval int      `json:"update_interval"`
}

// AppConfig représente la configuration de l'application
type AppConfig struct {
	AppName    string         `json:"app_name"`
	Version    string         `json:"version"`
	Owner      string         `json:"owner"`
	Theme      string         `json:"theme"`
	Language   string         `json:"language"`
	LogLevel   string         `json:"log_level"`
	MaxLogSize int            `json:"max_log_size"`
	LogBackups int            `json:"log_backups"`
	Database   DatabaseConfig `json:"database"`
}

// SearchFilter représente un filtre de recherche
type SearchFilter struct {
	Query       string      `json:"query"`
	Type        string      `json:"type"`
	ScannerType ScannerType `json:"scanner_type"`
	Country     string      `json:"country"`
	ISP         string      `json:"isp"`
	RiskLevel   string      `json:"risk_level"`
	DateFrom    time.Time   `json:"date_from"`
	DateTo      time.Time   `json:"date_to"`
}

// LogLevel représente le niveau de log
type LogLevel string

const (
	LogLevelDebug    LogLevel = "DEBUG"
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarning  LogLevel = "WARNING"
	LogLevelError    LogLevel = "ERROR"
	LogLevelCritical LogLevel = "CRITICAL"
)

// LogEntry représente une entrée de log
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}
