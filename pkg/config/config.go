package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the global configuration for infra-core
type Config struct {
	Gate         GateConfig         `yaml:"gate" json:"gate"`
	Console      ConsoleConfig      `yaml:"console" json:"console"`
	Orchestrator OrchestratorConfig `yaml:"orchestrator" json:"orchestrator"`
	Probe        ProbeMonitorConfig `yaml:"probe" json:"probe"`
	Snap         SnapConfig         `yaml:"snap" json:"snap"`
}

type LogConfig struct {
	Level   string `yaml:"level" json:"level"`
	Console bool   `yaml:"console" json:"console"`
	File    string `yaml:"file" json:"file"`
}

type PortsConfig struct {
	HTTP  int `yaml:"http" json:"http"`
	HTTPS int `yaml:"https" json:"https"`
}

type ACMEConfig struct {
	DirectoryURL  string `yaml:"directory_url" json:"directory_url"`
	Email         string `yaml:"email" json:"email"`
	CacheDir      string `yaml:"cache_dir" json:"cache_dir"`
	ChallengeType string `yaml:"challenge_type" json:"challenge_type"`
	Enabled       bool   `yaml:"enabled" json:"enabled"`
}

type GateConfig struct {
	Host  string      `yaml:"host" json:"host"`
	Ports PortsConfig `yaml:"ports" json:"ports"`
	Logs  LogConfig   `yaml:"logs" json:"logs"`
	ACME  ACMEConfig  `yaml:"acme" json:"acme"`
}

type DatabaseConfig struct {
	Path    string `yaml:"path" json:"path"`
	WALMode bool   `yaml:"wal_mode" json:"wal_mode"`
	Timeout string `yaml:"timeout" json:"timeout"`
}

type JWTConfig struct {
	Secret       string `yaml:"secret" json:"secret"`
	ExpiresHours int    `yaml:"expires_hours" json:"expires_hours"`
}

type SessionConfig struct {
	TimeoutMinutes int `yaml:"timeout_minutes" json:"timeout_minutes"`
}

type AuthConfig struct {
	JWT     JWTConfig     `yaml:"jwt" json:"jwt"`
	Session SessionConfig `yaml:"session" json:"session"`
}

type CORSConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Origins []string `yaml:"origins" json:"origins"`
	Methods []string `yaml:"methods" json:"methods"`
	Headers []string `yaml:"headers" json:"headers"`
}

type ConsoleConfig struct {
	Host     string         `yaml:"host" json:"host"`
	Port     int            `yaml:"port" json:"port"`
	Logs     LogConfig      `yaml:"logs" json:"logs"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Auth     AuthConfig     `yaml:"auth" json:"auth"`
	CORS     CORSConfig     `yaml:"cors" json:"cors"`
}

type OrchestratorConfig struct {
	Port                int    `yaml:"port" json:"port"`
	NodeName            string `yaml:"node_name" json:"node_name"`
	ClusterMode         bool   `yaml:"cluster_mode" json:"cluster_mode"`
	HealthCheckInterval string `yaml:"health_check_interval" json:"health_check_interval"`
	ResourceMonitoring  bool   `yaml:"resource_monitoring" json:"resource_monitoring"`
	DefaultReplicas     int    `yaml:"default_replicas" json:"default_replicas"`
	MaxDeployments      int    `yaml:"max_deployments" json:"max_deployments"`
	EnableMetrics       bool   `yaml:"enable_metrics" json:"enable_metrics"`
}

type ProbeMonitorConfig struct {
	Port              int    `yaml:"port" json:"port"`
	CheckInterval     string `yaml:"check_interval" json:"check_interval"`
	AlertInterval     string `yaml:"alert_interval" json:"alert_interval"`
	CleanupInterval   string `yaml:"cleanup_interval" json:"cleanup_interval"`
	ResultRetention   string `yaml:"result_retention" json:"result_retention"`
	AlertRetention    string `yaml:"alert_retention" json:"alert_retention"`
	EnableNotifications bool `yaml:"enable_notifications" json:"enable_notifications"`
	MaxConcurrentProbes int  `yaml:"max_concurrent_probes" json:"max_concurrent_probes"`
}

type SnapConfig struct {
	Port        int    `yaml:"port" json:"port"`
	RepoDir     string `yaml:"repo_dir" json:"repo_dir"`
	TempDir     string `yaml:"temp_dir" json:"temp_dir"`
	MaxParallel int    `yaml:"max_parallel" json:"max_parallel"`
	RateLimit   string `yaml:"rate_limit" json:"rate_limit"`
	ScrubInterval string `yaml:"scrub_interval" json:"scrub_interval"`
	DefaultRetention struct {
		Daily   int `yaml:"daily" json:"daily"`
		Weekly  int `yaml:"weekly" json:"weekly"`
		Monthly int `yaml:"monthly" json:"monthly"`
	} `yaml:"default_retention" json:"default_retention"`
}

// Global configuration instance
var globalConfig *Config

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	environment := os.Getenv("INFRA_CORE_ENV")
	if environment == "" {
		environment = "development"
	}
	
	// Determine config file path
	configPath := fmt.Sprintf("./configs/%s.yaml", environment)

	config := &Config{}

	// Load from file if exists
	if fileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	} else {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Override with environment variables
	overrideWithEnv(config)

	// Auto-generate JWT secret if empty
	if config.Console.Auth.JWT.Secret == "" && environment != "production" {
		config.Console.Auth.JWT.Secret = generateRandomSecret(32)
	}

	// Validate configuration
	if err := validate(config, environment); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	globalConfig = config
	return config, nil
}

// Get returns the global configuration instance
func Get() *Config {
	if globalConfig == nil {
		panic("configuration not loaded, call Load() first")
	}
	return globalConfig
}

// overrideWithEnv overrides configuration with environment variables
func overrideWithEnv(config *Config) {
	// Gate configuration
	if val := os.Getenv("INFRA_CORE_GATE_HOST"); val != "" {
		config.Gate.Host = val
	}
	if val := os.Getenv("INFRA_CORE_GATE_HTTP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Gate.Ports.HTTP = port
		}
	}
	if val := os.Getenv("INFRA_CORE_GATE_HTTPS_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Gate.Ports.HTTPS = port
		}
	}
	if val := os.Getenv("INFRA_CORE_ACME_EMAIL"); val != "" {
		config.Gate.ACME.Email = val
	}
	if val := os.Getenv("INFRA_CORE_ACME_ENABLED"); val != "" {
		config.Gate.ACME.Enabled = strings.ToLower(val) == "true"
	}

	// Console configuration
	if val := os.Getenv("INFRA_CORE_CONSOLE_HOST"); val != "" {
		config.Console.Host = val
	}
	if val := os.Getenv("INFRA_CORE_CONSOLE_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Console.Port = port
		}
	}
	if val := os.Getenv("INFRA_CORE_JWT_SECRET"); val != "" {
		config.Console.Auth.JWT.Secret = val
	}
	if val := os.Getenv("INFRA_CORE_DB_PATH"); val != "" {
		config.Console.Database.Path = val
	}

	// Orchestrator configuration
	if val := os.Getenv("INFRA_CORE_ORCH_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Orchestrator.Port = port
		}
	}
	if val := os.Getenv("INFRA_CORE_ORCH_NODE_NAME"); val != "" {
		config.Orchestrator.NodeName = val
	}
	if val := os.Getenv("INFRA_CORE_ORCH_CLUSTER_MODE"); val != "" {
		config.Orchestrator.ClusterMode = strings.ToLower(val) == "true"
	}

	// Probe configuration
	if val := os.Getenv("INFRA_CORE_PROBE_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Probe.Port = port
		}
	}
	if val := os.Getenv("INFRA_CORE_PROBE_CHECK_INTERVAL"); val != "" {
		config.Probe.CheckInterval = val
	}
	if val := os.Getenv("INFRA_CORE_PROBE_ENABLE_NOTIFICATIONS"); val != "" {
		config.Probe.EnableNotifications = strings.ToLower(val) == "true"
	}

	// Snap configuration
	if val := os.Getenv("INFRA_CORE_SNAP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Snap.Port = port
		}
	}
	if val := os.Getenv("INFRA_CORE_SNAP_REPO_DIR"); val != "" {
		config.Snap.RepoDir = val
	}
	if val := os.Getenv("INFRA_CORE_SNAP_TEMP_DIR"); val != "" {
		config.Snap.TempDir = val
	}
}

// validate validates the configuration
func validate(config *Config, environment string) error {
	// Validate Gate config
	if config.Gate.Host == "" {
		return fmt.Errorf("gate.host cannot be empty")
	}
	if config.Gate.Ports.HTTP <= 0 || config.Gate.Ports.HTTP > 65535 {
		return fmt.Errorf("invalid gate.ports.http: %d", config.Gate.Ports.HTTP)
	}
	if config.Gate.Ports.HTTPS <= 0 || config.Gate.Ports.HTTPS > 65535 {
		return fmt.Errorf("invalid gate.ports.https: %d", config.Gate.Ports.HTTPS)
	}

	// Validate Console config
	if config.Console.Host == "" {
		return fmt.Errorf("console.host cannot be empty")
	}
	if config.Console.Port <= 0 || config.Console.Port > 65535 {
		return fmt.Errorf("invalid console.port: %d", config.Console.Port)
	}
	if config.Console.Database.Path == "" {
		return fmt.Errorf("console.database.path cannot be empty")
	}

	// Validate Orchestrator config
	if config.Orchestrator.Port <= 0 || config.Orchestrator.Port > 65535 {
		return fmt.Errorf("invalid orchestrator.port: %d", config.Orchestrator.Port)
	}

	// Validate Probe config
	if config.Probe.Port <= 0 || config.Probe.Port > 65535 {
		return fmt.Errorf("invalid probe.port: %d", config.Probe.Port)
	}

	// Validate Snap config
	if config.Snap.Port <= 0 || config.Snap.Port > 65535 {
		return fmt.Errorf("invalid snap.port: %d", config.Snap.Port)
	}
	if config.Snap.RepoDir == "" {
		return fmt.Errorf("snap.repo_dir cannot be empty")
	}
	if config.Snap.TempDir == "" {
		return fmt.Errorf("snap.temp_dir cannot be empty")
	}

	// JWT secret is required in production
	if environment == "production" && config.Console.Auth.JWT.Secret == "" {
		return fmt.Errorf("console.auth.jwt.secret is required in production environment")
	}

	return nil
}

// generateRandomSecret generates a random secret for JWT
func generateRandomSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[len(charset)/2] // Simple fallback
	}
	return string(b)
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
