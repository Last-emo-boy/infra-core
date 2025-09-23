package database

import (
	"encoding/json"
	"time"
)

// User represents a system user
type User struct {
	ID           int        `db:"id" json:"id"`
	Username     string     `db:"username" json:"username"`
	Email        string     `db:"email" json:"email"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         string     `db:"role" json:"role"`
	TOTPSecret   *string    `db:"totp_secret" json:"-"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	LastLogin    *time.Time `db:"last_login" json:"last_login"`
}

// Service represents a deployed service
type Service struct {
	ID          string            `db:"id" json:"id"`
	Name        string            `db:"name" json:"name"`
	Image       string            `db:"image" json:"image"`
	Port        int               `db:"port" json:"port"`
	Replicas    int               `db:"replicas" json:"replicas"`
	Status      string            `db:"status" json:"status"`
	Environment map[string]string `db:"environment" json:"environment"`
	Command     []string          `db:"command" json:"command"`
	Args        []string          `db:"args" json:"args"`
	YAMLConfig  string            `db:"yaml_config" json:"yaml_config"`
	Version     int               `db:"version" json:"version"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}

// MarshalEnvironment converts environment map to JSON string for database storage
func (s *Service) MarshalEnvironment() (string, error) {
	if s.Environment == nil {
		return "{}", nil
	}
	data, err := json.Marshal(s.Environment)
	return string(data), err
}

// UnmarshalEnvironment converts JSON string to environment map
func (s *Service) UnmarshalEnvironment(data string) error {
	if data == "" {
		s.Environment = make(map[string]string)
		return nil
	}
	return json.Unmarshal([]byte(data), &s.Environment)
}

// MarshalCommand converts command slice to JSON string for database storage
func (s *Service) MarshalCommand() (string, error) {
	if s.Command == nil {
		return "[]", nil
	}
	data, err := json.Marshal(s.Command)
	return string(data), err
}

// UnmarshalCommand converts JSON string to command slice
func (s *Service) UnmarshalCommand(data string) error {
	if data == "" {
		s.Command = []string{}
		return nil
	}
	return json.Unmarshal([]byte(data), &s.Command)
}

// MarshalArgs converts args slice to JSON string for database storage
func (s *Service) MarshalArgs() (string, error) {
	if s.Args == nil {
		return "[]", nil
	}
	data, err := json.Marshal(s.Args)
	return string(data), err
}

// UnmarshalArgs converts JSON string to args slice
func (s *Service) UnmarshalArgs(data string) error {
	if data == "" {
		s.Args = []string{}
		return nil
	}
	return json.Unmarshal([]byte(data), &s.Args)
}

// Deployment represents a service deployment
type Deployment struct {
	ID           string     `db:"id" json:"id"`
	ServiceID    string     `db:"service_id" json:"service_id"`
	Version      int        `db:"version" json:"version"`
	Status       string     `db:"status" json:"status"`
	StartedAt    time.Time  `db:"started_at" json:"started_at"`
	FinishedAt   *time.Time `db:"finished_at" json:"finished_at"`
	ErrorMessage *string    `db:"error_message" json:"error_message"`
}

// Route represents a routing rule
type Route struct {
	ID                string    `db:"id" json:"id"`
	Host              string    `db:"host" json:"host"`
	PathPrefix        string    `db:"path_prefix" json:"path_prefix"`
	UpstreamServiceID *string   `db:"upstream_service_id" json:"upstream_service_id"`
	UpstreamURL       *string   `db:"upstream_url" json:"upstream_url"`
	TLSCertID         *string   `db:"tls_cert_id" json:"tls_cert_id"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

// Certificate represents a TLS certificate
type Certificate struct {
	ID         string    `db:"id" json:"id"`
	Domain     string    `db:"domain" json:"domain"`
	NotBefore  time.Time `db:"not_before" json:"not_before"`
	NotAfter   time.Time `db:"not_after" json:"not_after"`
	CertPath   string    `db:"cert_path" json:"cert_path"`
	KeyPath    string    `db:"key_path" json:"key_path"`
	IssuerPath *string   `db:"issuer_path" json:"issuer_path"`
	Status     string    `db:"status" json:"status"`
	AutoRenew  bool      `db:"auto_renew" json:"auto_renew"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// Metric represents a time series data point
type Metric struct {
	ID          int       `db:"id" json:"id"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	ScopeType   string    `db:"scope_type" json:"scope_type"`
	ScopeID     string    `db:"scope_id" json:"scope_id"`
	MetricName  string    `db:"metric_name" json:"metric_name"`
	MetricValue float64   `db:"metric_value" json:"metric_value"`
	Labels      *string   `db:"labels" json:"labels"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// LogIndex represents log file index information
type LogIndex struct {
	ID             int       `db:"id" json:"id"`
	ServiceID      string    `db:"service_id" json:"service_id"`
	LogFile        string    `db:"log_file" json:"log_file"`
	StartTimestamp time.Time `db:"start_timestamp" json:"start_timestamp"`
	EndTimestamp   time.Time `db:"end_timestamp" json:"end_timestamp"`
	OffsetStart    int64     `db:"offset_start" json:"offset_start"`
	OffsetEnd      int64     `db:"offset_end" json:"offset_end"`
	LineCount      int       `db:"line_count" json:"line_count"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// Snapshot represents a backup snapshot
type Snapshot struct {
	ID           string    `db:"id" json:"id"`
	PlanID       string    `db:"plan_id" json:"plan_id"`
	Timestamp    time.Time `db:"timestamp" json:"timestamp"`
	ManifestPath string    `db:"manifest_path" json:"manifest_path"`
	SizeBytes    int64     `db:"size_bytes" json:"size_bytes"`
	Kind         string    `db:"kind" json:"kind"`
	Status       string    `db:"status" json:"status"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// SnapPlan represents a backup plan
type SnapPlan struct {
	ID             string    `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	CronExpression string    `db:"cron_expression" json:"cron_expression"`
	Paths          string    `db:"paths" json:"paths"` // JSON array
	KeepDaily      int       `db:"keep_daily" json:"keep_daily"`
	KeepWeekly     int       `db:"keep_weekly" json:"keep_weekly"`
	KeepMonthly    int       `db:"keep_monthly" json:"keep_monthly"`
	Enabled        bool      `db:"enabled" json:"enabled"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           int       `db:"id" json:"id"`
	UserID       *int      `db:"user_id" json:"user_id"`
	Action       string    `db:"action" json:"action"`
	ResourceType string    `db:"resource_type" json:"resource_type"`
	ResourceID   *string   `db:"resource_id" json:"resource_id"`
	Details      *string   `db:"details" json:"details"`
	IPAddress    *string   `db:"ip_address" json:"ip_address"`
	UserAgent    *string   `db:"user_agent" json:"user_agent"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// RegisteredService represents a service registered with the SSO gateway
type RegisteredService struct {
	ID           string    `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	DisplayName  string    `db:"display_name" json:"display_name"`
	Description  *string   `db:"description" json:"description"`
	ServiceURL   string    `db:"service_url" json:"service_url"`
	CallbackURL  *string   `db:"callback_url" json:"callback_url"`
	Icon         *string   `db:"icon" json:"icon"`
	Category     string    `db:"category" json:"category"`
	IsPublic     bool      `db:"is_public" json:"is_public"`
	RequiredRole string    `db:"required_role" json:"required_role"`
	Status       string    `db:"status" json:"status"` // active, inactive, maintenance
	HealthURL    *string   `db:"health_url" json:"health_url"`
	LastHealthy  *time.Time `db:"last_healthy" json:"last_healthy"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// SSOSession represents an SSO session
type SSOSession struct {
	ID           string    `db:"id" json:"id"`
	UserID       int       `db:"user_id" json:"user_id"`
	TokenHash    string    `db:"token_hash" json:"-"`
	ExpiresAt    time.Time `db:"expires_at" json:"expires_at"`
	IPAddress    string    `db:"ip_address" json:"ip_address"`
	UserAgent    string    `db:"user_agent" json:"user_agent"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	LastUsed     time.Time `db:"last_used" json:"last_used"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// UserServicePermission represents user permissions for specific services
type UserServicePermission struct {
	ID        int       `db:"id" json:"id"`
	UserID    int       `db:"user_id" json:"user_id"`
	ServiceID string    `db:"service_id" json:"service_id"`
	CanAccess bool      `db:"can_access" json:"can_access"`
	GrantedBy int       `db:"granted_by" json:"granted_by"`
	GrantedAt time.Time `db:"granted_at" json:"granted_at"`
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at"`
}

// ServiceHealthCheck represents health check results for registered services
type ServiceHealthCheck struct {
	ID          int       `db:"id" json:"id"`
	ServiceID   string    `db:"service_id" json:"service_id"`
	IsHealthy   bool      `db:"is_healthy" json:"is_healthy"`
	ResponseTime int       `db:"response_time" json:"response_time"` // in milliseconds
	ErrorMessage *string   `db:"error_message" json:"error_message"`
	CheckedAt   time.Time `db:"checked_at" json:"checked_at"`
}
