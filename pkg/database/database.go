package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

// DB represents the database connection
type DB struct {
	*sqlx.DB
	config *config.Config
}

// NewDB creates a new database connection
func NewDB(cfg *config.Config) (*DB, error) {
	dbPath := cfg.Console.Database.Path

	// Handle special case for in-memory database
	if dbPath == ":memory:" {
		// Connect directly to in-memory database
		db, err := sqlx.Connect("sqlite", ":memory:")
		if err != nil {
			return nil, fmt.Errorf("failed to connect to in-memory database: %w", err)
		}

		// Create database instance
		database := &DB{
			DB:     db,
			config: cfg,
		}

		// Initialize schema
		if err := database.InitSchema(); err != nil {
			return nil, fmt.Errorf("failed to initialize schema: %w", err)
		}

		return database, nil
	}

	// Ensure data directory exists for file-based database
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Build connection string
	connStr := dbPath
	if cfg.Console.Database.WALMode {
		connStr += "?_journal_mode=WAL&_sync=NORMAL&_cache_size=1000&_foreign_keys=ON"
	}

	// Open database
	db, err := sqlx.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool with reasonable defaults
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dbWrapper := &DB{
		DB:     db,
		config: cfg,
	}

	// Initialize schema
	if err := dbWrapper.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return dbWrapper, nil
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user', -- admin, user
		totp_secret TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME
	);

	-- Services table
	CREATE TABLE IF NOT EXISTS services (
		id TEXT PRIMARY KEY, -- UUID
		name TEXT UNIQUE NOT NULL,
		image TEXT NOT NULL,
		port INTEGER NOT NULL DEFAULT 8080,
		replicas INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'stopped', -- running, stopped, error
		environment TEXT, -- JSON string for environment variables
		command TEXT, -- JSON string for command array
		args TEXT, -- JSON string for args array
		yaml_config TEXT,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Deployments table
	CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY, -- UUID
		service_id TEXT NOT NULL,
		version INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending', -- pending, running, success, failed
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		error_message TEXT,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
	);

	-- Routes table
	CREATE TABLE IF NOT EXISTS routes (
		id TEXT PRIMARY KEY, -- UUID
		host TEXT NOT NULL,
		path_prefix TEXT NOT NULL DEFAULT '/',
		upstream_service_id TEXT,
		upstream_url TEXT,
		tls_cert_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (upstream_service_id) REFERENCES services(id) ON DELETE SET NULL,
		FOREIGN KEY (tls_cert_id) REFERENCES certificates(id) ON DELETE SET NULL
	);

	-- Certificates table
	CREATE TABLE IF NOT EXISTS certificates (
		id TEXT PRIMARY KEY, -- UUID
		domain TEXT UNIQUE NOT NULL,
		not_before DATETIME NOT NULL,
		not_after DATETIME NOT NULL,
		cert_path TEXT NOT NULL,
		key_path TEXT NOT NULL,
		issuer_path TEXT,
		status TEXT NOT NULL DEFAULT 'valid', -- valid, expired, revoked
		auto_renew BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Metrics table (time series data)
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		scope_type TEXT NOT NULL, -- host, service, route
		scope_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		metric_value REAL NOT NULL,
		labels TEXT, -- JSON format
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Logs index table
	CREATE TABLE IF NOT EXISTS logs_index (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id TEXT NOT NULL,
		log_file TEXT NOT NULL,
		start_timestamp DATETIME NOT NULL,
		end_timestamp DATETIME NOT NULL,
		offset_start INTEGER NOT NULL,
		offset_end INTEGER NOT NULL,
		line_count INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
	);

	-- Snapshots table
	CREATE TABLE IF NOT EXISTS snapshots (
		id TEXT PRIMARY KEY, -- UUID
		plan_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		manifest_path TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		kind TEXT NOT NULL DEFAULT 'incremental', -- full, incremental
		status TEXT NOT NULL DEFAULT 'creating', -- creating, completed, failed
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (plan_id) REFERENCES snap_plans(id) ON DELETE CASCADE
	);

	-- Snapshot plans table
	CREATE TABLE IF NOT EXISTS snap_plans (
		id TEXT PRIMARY KEY, -- UUID
		name TEXT UNIQUE NOT NULL,
		cron_expression TEXT NOT NULL,
		paths TEXT NOT NULL, -- JSON array
		keep_daily INTEGER NOT NULL DEFAULT 7,
		keep_weekly INTEGER NOT NULL DEFAULT 4,
		keep_monthly INTEGER NOT NULL DEFAULT 3,
		enabled BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Audit log table
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT,
		details TEXT, -- JSON format
		ip_address TEXT,
		user_agent TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
	);

	-- Registered services table (for SSO gateway)
	CREATE TABLE IF NOT EXISTS registered_services (
		id TEXT PRIMARY KEY, -- UUID
		name TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT,
		service_url TEXT NOT NULL,
		callback_url TEXT,
		icon TEXT,
		category TEXT NOT NULL DEFAULT 'other', -- web, api, admin, monitoring, other
		is_public BOOLEAN DEFAULT FALSE,
		required_role TEXT NOT NULL DEFAULT 'user', -- user, admin
		status TEXT NOT NULL DEFAULT 'active', -- active, inactive, maintenance
		health_url TEXT,
		last_healthy DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- SSO sessions table
	CREATE TABLE IF NOT EXISTS sso_sessions (
		id TEXT PRIMARY KEY, -- UUID
		user_id INTEGER NOT NULL,
		token_hash TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		ip_address TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- User service permissions table
	CREATE TABLE IF NOT EXISTS user_service_permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		service_id TEXT NOT NULL,
		can_access BOOLEAN DEFAULT TRUE,
		granted_by INTEGER NOT NULL,
		granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (service_id) REFERENCES registered_services(id) ON DELETE CASCADE,
		FOREIGN KEY (granted_by) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, service_id)
	);

	-- Service health checks table
	CREATE TABLE IF NOT EXISTS service_health_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id TEXT NOT NULL,
		is_healthy BOOLEAN NOT NULL,
		response_time INTEGER, -- milliseconds
		error_message TEXT,
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (service_id) REFERENCES registered_services(id) ON DELETE CASCADE
	);

	-- Create indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
	CREATE INDEX IF NOT EXISTS idx_deployments_service_id ON deployments(service_id);
	CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
	CREATE INDEX IF NOT EXISTS idx_routes_host ON routes(host);
	CREATE INDEX IF NOT EXISTS idx_routes_path_prefix ON routes(path_prefix);
	CREATE INDEX IF NOT EXISTS idx_certificates_domain ON certificates(domain);
	CREATE INDEX IF NOT EXISTS idx_certificates_not_after ON certificates(not_after);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
	CREATE INDEX IF NOT EXISTS idx_metrics_scope ON metrics(scope_type, scope_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(metric_name);
	CREATE INDEX IF NOT EXISTS idx_logs_service_timestamp ON logs_index(service_id, start_timestamp);
	CREATE INDEX IF NOT EXISTS idx_snapshots_plan_timestamp ON snapshots(plan_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_timestamp ON audit_logs(user_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_registered_services_status ON registered_services(status);
	CREATE INDEX IF NOT EXISTS idx_registered_services_category ON registered_services(category);
	CREATE INDEX IF NOT EXISTS idx_registered_services_role ON registered_services(required_role);
	CREATE INDEX IF NOT EXISTS idx_sso_sessions_user_id ON sso_sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sso_sessions_token_hash ON sso_sessions(token_hash);
	CREATE INDEX IF NOT EXISTS idx_sso_sessions_expires_at ON sso_sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_user_service_permissions_user_id ON user_service_permissions(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_service_permissions_service_id ON user_service_permissions(service_id);
	CREATE INDEX IF NOT EXISTS idx_service_health_checks_service_id ON service_health_checks(service_id);
	CREATE INDEX IF NOT EXISTS idx_service_health_checks_checked_at ON service_health_checks(checked_at);

	-- Create triggers for updated_at timestamps
	CREATE TRIGGER IF NOT EXISTS update_users_timestamp 
		AFTER UPDATE ON users
		BEGIN
			UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TRIGGER IF NOT EXISTS update_services_timestamp 
		AFTER UPDATE ON services
		BEGIN
			UPDATE services SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TRIGGER IF NOT EXISTS update_routes_timestamp 
		AFTER UPDATE ON routes
		BEGIN
			UPDATE routes SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TRIGGER IF NOT EXISTS update_certificates_timestamp 
		AFTER UPDATE ON certificates
		BEGIN
			UPDATE certificates SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TRIGGER IF NOT EXISTS update_snap_plans_timestamp 
		AFTER UPDATE ON snap_plans
		BEGIN
			UPDATE snap_plans SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TRIGGER IF NOT EXISTS update_registered_services_timestamp 
		AFTER UPDATE ON registered_services
		BEGIN
			UPDATE registered_services SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck() error {
	var result int
	err := db.Get(&result, "SELECT 1")
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// GetStats returns database statistics
func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get table counts
	tables := []string{"users", "services", "deployments", "routes", "certificates", "metrics", "logs_index", "snapshots", "snap_plans", "audit_logs", "registered_services", "sso_sessions", "user_service_permissions", "service_health_checks"}

	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.Get(&count, query); err != nil {
			return nil, fmt.Errorf("failed to count %s: %w", table, err)
		}
		stats[table+"_count"] = count
	}

	// Get database size
	var pages, pageSize int
	if err := db.Get(&pages, "PRAGMA page_count"); err == nil {
		if err := db.Get(&pageSize, "PRAGMA page_size"); err == nil {
			stats["database_size_bytes"] = pages * pageSize
		}
	}

	// Get WAL mode status
	var walMode string
	if err := db.Get(&walMode, "PRAGMA journal_mode"); err == nil {
		stats["journal_mode"] = walMode
	}

	return stats, nil
}

// UserRepository returns a new user repository
func (db *DB) UserRepository() *UserRepository {
	return NewUserRepository(db)
}

// ServiceRepository returns a new service repository
func (db *DB) ServiceRepository() *ServiceRepository {
	return NewServiceRepository(db)
}

// RouteRepository returns a new route repository
func (db *DB) RouteRepository() *RouteRepository {
	return NewRouteRepository(db)
}

// MetricRepository returns a new metric repository
func (db *DB) MetricRepository() *MetricRepository {
	return NewMetricRepository(db)
}

// RegisteredServiceRepository returns a new registered service repository
func (db *DB) RegisteredServiceRepository() *RegisteredServiceRepository {
	return NewRegisteredServiceRepository(db)
}

// SSOSessionRepository returns a new SSO session repository
func (db *DB) SSOSessionRepository() *SSOSessionRepository {
	return NewSSOSessionRepository(db)
}

// UserServicePermissionRepository returns a new user service permission repository
func (db *DB) UserServicePermissionRepository() *UserServicePermissionRepository {
	return NewUserServicePermissionRepository(db)
}

// ServiceHealthCheckRepository returns a new service health check repository
func (db *DB) ServiceHealthCheckRepository() *ServiceHealthCheckRepository {
	return NewServiceHealthCheckRepository(db)
}
