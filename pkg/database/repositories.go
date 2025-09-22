package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserRepository provides database operations for users
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, totp_secret)
		VALUES (:username, :email, :password_hash, :role, :totp_secret)
	`
	result, err := r.db.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	user.ID = int(id)
	return nil
}

// GetByID gets a user by ID
func (r *UserRepository) GetByID(id int) (*User, error) {
	var user User
	query := "SELECT * FROM users WHERE id = ?"
	err := r.db.Get(&user, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

// GetByUsername gets a user by username
func (r *UserRepository) GetByUsername(username string) (*User, error) {
	var user User
	query := "SELECT * FROM users WHERE username = ?"
	err := r.db.Get(&user, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *User) error {
	query := `
		UPDATE users 
		SET username = :username, email = :email, password_hash = :password_hash, 
		    role = :role, totp_secret = :totp_secret, last_login = :last_login
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateLastLogin(userID int) error {
	query := "UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(userID int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// List lists all users
func (r *UserRepository) List() ([]*User, error) {
	var users []*User
	query := "SELECT * FROM users ORDER BY created_at DESC"
	err := r.db.Select(&users, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// ServiceRepository provides database operations for services
type ServiceRepository struct {
	db *DB
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create creates a new service
func (r *ServiceRepository) Create(service *Service) error {
	if service.ID == "" {
		service.ID = uuid.New().String()
	}

	// Serialize complex fields
	envJSON, err := service.MarshalEnvironment()
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	cmdJSON, err := service.MarshalCommand()
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	argsJSON, err := service.MarshalArgs()
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	query := `
		INSERT INTO services (id, name, image, port, replicas, status, environment, command, args, yaml_config, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query, service.ID, service.Name, service.Image, service.Port, service.Replicas,
		service.Status, envJSON, cmdJSON, argsJSON, service.YAMLConfig, service.Version)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	return nil
}

// GetByID gets a service by ID
func (r *ServiceRepository) GetByID(id string) (*Service, error) {
	var service Service
	var envJSON, cmdJSON, argsJSON string

	query := `SELECT id, name, image, port, replicas, status, environment, command, args, 
		yaml_config, version, created_at, updated_at FROM services WHERE id = ?`
	row := r.db.QueryRow(query, id)

	err := row.Scan(&service.ID, &service.Name, &service.Image, &service.Port, &service.Replicas,
		&service.Status, &envJSON, &cmdJSON, &argsJSON, &service.YAMLConfig, &service.Version,
		&service.CreatedAt, &service.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get service by ID: %w", err)
	}

	// Deserialize JSON fields
	if err := service.UnmarshalEnvironment(envJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}
	if err := service.UnmarshalCommand(cmdJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %w", err)
	}
	if err := service.UnmarshalArgs(argsJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return &service, nil
}

// GetByName gets a service by name
func (r *ServiceRepository) GetByName(name string) (*Service, error) {
	var service Service
	var envJSON, cmdJSON, argsJSON string

	query := `SELECT id, name, image, port, replicas, status, environment, command, args, 
		yaml_config, version, created_at, updated_at FROM services WHERE name = ?`
	row := r.db.QueryRow(query, name)

	err := row.Scan(&service.ID, &service.Name, &service.Image, &service.Port, &service.Replicas,
		&service.Status, &envJSON, &cmdJSON, &argsJSON, &service.YAMLConfig, &service.Version,
		&service.CreatedAt, &service.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get service by name: %w", err)
	}

	// Deserialize JSON fields
	if err := service.UnmarshalEnvironment(envJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}
	if err := service.UnmarshalCommand(cmdJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %w", err)
	}
	if err := service.UnmarshalArgs(argsJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return &service, nil
}

// Update updates a service
func (r *ServiceRepository) Update(service *Service) error {
	service.Version++

	// Serialize complex fields
	envJSON, err := service.MarshalEnvironment()
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	cmdJSON, err := service.MarshalCommand()
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	argsJSON, err := service.MarshalArgs()
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	query := `
		UPDATE services 
		SET name = ?, image = ?, port = ?, replicas = ?, status = ?, 
			environment = ?, command = ?, args = ?, yaml_config = ?, version = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err = r.db.Exec(query, service.Name, service.Image, service.Port, service.Replicas, service.Status,
		envJSON, cmdJSON, argsJSON, service.YAMLConfig, service.Version, service.ID)
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}
	return nil
}

// List lists all services
func (r *ServiceRepository) List() ([]*Service, error) {
	query := `SELECT id, name, image, port, replicas, status, environment, command, args, 
		yaml_config, version, created_at, updated_at FROM services ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var service Service
		var envJSON, cmdJSON, argsJSON string

		err := rows.Scan(&service.ID, &service.Name, &service.Image, &service.Port, &service.Replicas,
			&service.Status, &envJSON, &cmdJSON, &argsJSON, &service.YAMLConfig, &service.Version,
			&service.CreatedAt, &service.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}

		// Deserialize JSON fields
		if err := service.UnmarshalEnvironment(envJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
		}
		if err := service.UnmarshalCommand(cmdJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command: %w", err)
		}
		if err := service.UnmarshalArgs(argsJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", err)
		}

		services = append(services, &service)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating services: %w", err)
	}

	return services, nil
}

// Delete deletes a service
func (r *ServiceRepository) Delete(id string) error {
	query := "DELETE FROM services WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}
	return nil
}

// RouteRepository provides database operations for routes
type RouteRepository struct {
	db *DB
}

// NewRouteRepository creates a new route repository
func NewRouteRepository(db *DB) *RouteRepository {
	return &RouteRepository{db: db}
}

// Create creates a new route
func (r *RouteRepository) Create(route *Route) error {
	if route.ID == "" {
		route.ID = uuid.New().String()
	}

	query := `
		INSERT INTO routes (id, host, path_prefix, upstream_service_id, upstream_url, tls_cert_id)
		VALUES (:id, :host, :path_prefix, :upstream_service_id, :upstream_url, :tls_cert_id)
	`
	_, err := r.db.NamedExec(query, route)
	if err != nil {
		return fmt.Errorf("failed to create route: %w", err)
	}
	return nil
}

// GetByID gets a route by ID
func (r *RouteRepository) GetByID(id string) (*Route, error) {
	var route Route
	query := "SELECT * FROM routes WHERE id = ?"
	err := r.db.Get(&route, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get route by ID: %w", err)
	}
	return &route, nil
}

// List lists all routes
func (r *RouteRepository) List() ([]*Route, error) {
	var routes []*Route
	query := "SELECT * FROM routes ORDER BY created_at DESC"
	err := r.db.Select(&routes, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	return routes, nil
}

// Update updates a route
func (r *RouteRepository) Update(route *Route) error {
	query := `
		UPDATE routes 
		SET host = :host, path_prefix = :path_prefix, upstream_service_id = :upstream_service_id, 
		    upstream_url = :upstream_url, tls_cert_id = :tls_cert_id
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, route)
	if err != nil {
		return fmt.Errorf("failed to update route: %w", err)
	}
	return nil
}

// Delete deletes a route
func (r *RouteRepository) Delete(id string) error {
	query := "DELETE FROM routes WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}

// MetricRepository provides database operations for metrics
type MetricRepository struct {
	db *DB
}

// NewMetricRepository creates a new metric repository
func NewMetricRepository(db *DB) *MetricRepository {
	return &MetricRepository{db: db}
}

// Insert inserts a new metric
func (r *MetricRepository) Insert(metric *Metric) error {
	query := `
		INSERT INTO metrics (timestamp, scope_type, scope_id, metric_name, metric_value, labels)
		VALUES (:timestamp, :scope_type, :scope_id, :metric_name, :metric_value, :labels)
	`
	_, err := r.db.NamedExec(query, metric)
	if err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}
	return nil
}

// Query queries metrics by time range and filters
func (r *MetricRepository) Query(scopeType, scopeID, metricName string, from, to time.Time, limit int) ([]*Metric, error) {
	var metrics []*Metric
	query := `
		SELECT * FROM metrics 
		WHERE scope_type = ? AND scope_id = ? AND metric_name = ? 
		  AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	err := r.db.Select(&metrics, query, scopeType, scopeID, metricName, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	return metrics, nil
}

// DeleteOld deletes old metrics beyond retention period
func (r *MetricRepository) DeleteOld(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	query := "DELETE FROM metrics WHERE timestamp < ?"
	_, err := r.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}
	return nil
}

// GetByService gets metrics for a specific service
func (r *MetricRepository) GetByService(serviceID string, limit int) ([]*Metric, error) {
	var metrics []*Metric
	query := `
		SELECT * FROM metrics 
		WHERE scope_type = 'service' AND scope_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	err := r.db.Select(&metrics, query, serviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by service: %w", err)
	}
	return metrics, nil
}

// GetRecent gets recent metrics across all services
func (r *MetricRepository) GetRecent(limit int) ([]*Metric, error) {
	var metrics []*Metric
	query := `
		SELECT * FROM metrics 
		ORDER BY timestamp DESC
		LIMIT ?
	`
	err := r.db.Select(&metrics, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent metrics: %w", err)
	}
	return metrics, nil
}

// AuditLogRepository provides database operations for audit logs
type AuditLogRepository struct {
	db *DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES (:user_id, :action, :resource_type, :resource_id, :details, :ip_address, :user_agent)
	`
	_, err := r.db.NamedExec(query, log)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// List lists audit logs with pagination
func (r *AuditLogRepository) List(limit, offset int) ([]*AuditLog, error) {
	var logs []*AuditLog
	query := "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?"
	err := r.db.Select(&logs, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return logs, nil
}
