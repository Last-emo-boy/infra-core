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

// Delete deletes a user account
func (r *UserRepository) Delete(userID int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// RegisteredServiceRepository provides database operations for registered services
type RegisteredServiceRepository struct {
	db *DB
}

// NewRegisteredServiceRepository creates a new registered service repository
func NewRegisteredServiceRepository(db *DB) *RegisteredServiceRepository {
	return &RegisteredServiceRepository{db: db}
}

// Create creates a new registered service
func (r *RegisteredServiceRepository) Create(service *RegisteredService) error {
	if service.ID == "" {
		service.ID = uuid.New().String()
	}
	
	query := `
		INSERT INTO registered_services (id, name, display_name, description, service_url, callback_url, icon, category, is_public, required_role, status, health_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, service.ID, service.Name, service.DisplayName, service.Description, service.ServiceURL, service.CallbackURL, service.Icon, service.Category, service.IsPublic, service.RequiredRole, service.Status, service.HealthURL)
	if err != nil {
		return fmt.Errorf("failed to create registered service: %w", err)
	}

	return nil
}

// GetByID gets a registered service by ID
func (r *RegisteredServiceRepository) GetByID(id string) (*RegisteredService, error) {
	var service RegisteredService
	query := `SELECT id, name, display_name, description, service_url, callback_url, icon, category, is_public, required_role, status, health_url, last_healthy, created_at, updated_at FROM registered_services WHERE id = ?`
	err := r.db.Get(&service, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered service: %w", err)
	}

	return &service, nil
}

// GetByName gets a registered service by name
func (r *RegisteredServiceRepository) GetByName(name string) (*RegisteredService, error) {
	var service RegisteredService
	query := `SELECT id, name, display_name, description, service_url, callback_url, icon, category, is_public, required_role, status, health_url, last_healthy, created_at, updated_at FROM registered_services WHERE name = ?`
	err := r.db.Get(&service, query, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered service: %w", err)
	}

	return &service, nil
}

// List lists all registered services
func (r *RegisteredServiceRepository) List() ([]*RegisteredService, error) {
	query := `SELECT id, name, display_name, description, service_url, callback_url, icon, category, is_public, required_role, status, health_url, last_healthy, created_at, updated_at FROM registered_services ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query registered services: %w", err)
	}
	defer rows.Close()

	var services []*RegisteredService
	for rows.Next() {
		var service RegisteredService
		err := rows.Scan(&service.ID, &service.Name, &service.DisplayName, &service.Description, &service.ServiceURL, &service.CallbackURL, &service.Icon, &service.Category, &service.IsPublic, &service.RequiredRole, &service.Status, &service.HealthURL, &service.LastHealthy, &service.CreatedAt, &service.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan registered service: %w", err)
		}
		services = append(services, &service)
	}

	return services, nil
}

// ListByCategory lists registered services by category
func (r *RegisteredServiceRepository) ListByCategory(category string) ([]*RegisteredService, error) {
	query := `SELECT id, name, display_name, description, service_url, callback_url, icon, category, is_public, required_role, status, health_url, last_healthy, created_at, updated_at FROM registered_services WHERE category = ? ORDER BY display_name`
	rows, err := r.db.Query(query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query registered services by category: %w", err)
	}
	defer rows.Close()

	var services []*RegisteredService
	for rows.Next() {
		var service RegisteredService
		err := rows.Scan(&service.ID, &service.Name, &service.DisplayName, &service.Description, &service.ServiceURL, &service.CallbackURL, &service.Icon, &service.Category, &service.IsPublic, &service.RequiredRole, &service.Status, &service.HealthURL, &service.LastHealthy, &service.CreatedAt, &service.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan registered service: %w", err)
		}
		services = append(services, &service)
	}

	return services, nil
}

// Update updates a registered service
func (r *RegisteredServiceRepository) Update(service *RegisteredService) error {
	query := `
		UPDATE registered_services 
		SET display_name = ?, description = ?, service_url = ?, callback_url = ?, icon = ?, category = ?, is_public = ?, required_role = ?, status = ?, health_url = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, service.DisplayName, service.Description, service.ServiceURL, service.CallbackURL, service.Icon, service.Category, service.IsPublic, service.RequiredRole, service.Status, service.HealthURL, service.ID)
	if err != nil {
		return fmt.Errorf("failed to update registered service: %w", err)
	}

	return nil
}

// UpdateHealthStatus updates the health status of a service
func (r *RegisteredServiceRepository) UpdateHealthStatus(serviceID string, isHealthy bool) error {
	var lastHealthy *time.Time
	if isHealthy {
		now := time.Now()
		lastHealthy = &now
	}
	
	query := `UPDATE registered_services SET last_healthy = ? WHERE id = ?`
	_, err := r.db.Exec(query, lastHealthy, serviceID)
	if err != nil {
		return fmt.Errorf("failed to update service health status: %w", err)
	}

	return nil
}

// Delete deletes a registered service
func (r *RegisteredServiceRepository) Delete(serviceID string) error {
	query := `DELETE FROM registered_services WHERE id = ?`
	_, err := r.db.Exec(query, serviceID)
	if err != nil {
		return fmt.Errorf("failed to delete registered service: %w", err)
	}

	return nil
}

// SSOSessionRepository provides database operations for SSO sessions
type SSOSessionRepository struct {
	db *DB
}

// NewSSOSessionRepository creates a new SSO session repository
func NewSSOSessionRepository(db *DB) *SSOSessionRepository {
	return &SSOSessionRepository{db: db}
}

// Create creates a new SSO session
func (r *SSOSessionRepository) Create(session *SSOSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	
	query := `
		INSERT INTO sso_sessions (id, user_id, token_hash, expires_at, ip_address, user_agent, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.IPAddress, session.UserAgent, session.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create SSO session: %w", err)
	}

	return nil
}

// GetByTokenHash gets an SSO session by token hash
func (r *SSOSessionRepository) GetByTokenHash(tokenHash string) (*SSOSession, error) {
	var session SSOSession
	query := `SELECT id, user_id, token_hash, expires_at, ip_address, user_agent, is_active, last_used, created_at FROM sso_sessions WHERE token_hash = ? AND is_active = TRUE AND expires_at > ?`
	err := r.db.Get(&session, query, tokenHash, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO session: %w", err)
	}

	return &session, nil
}

// UpdateLastUsed updates the last used timestamp for a session
func (r *SSOSessionRepository) UpdateLastUsed(sessionID string) error {
	query := `UPDATE sso_sessions SET last_used = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session last used: %w", err)
	}

	return nil
}

// Invalidate invalidates an SSO session
func (r *SSOSessionRepository) Invalidate(sessionID string) error {
	query := `UPDATE sso_sessions SET is_active = FALSE WHERE id = ?`
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to invalidate SSO session: %w", err)
	}

	return nil
}

// InvalidateUserSessions invalidates all sessions for a user
func (r *SSOSessionRepository) InvalidateUserSessions(userID int) error {
	query := `UPDATE sso_sessions SET is_active = FALSE WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (r *SSOSessionRepository) CleanupExpiredSessions() error {
	query := `DELETE FROM sso_sessions WHERE expires_at < ?`
	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}

// UserServicePermissionRepository provides database operations for user service permissions
type UserServicePermissionRepository struct {
	db *DB
}

// NewUserServicePermissionRepository creates a new user service permission repository
func NewUserServicePermissionRepository(db *DB) *UserServicePermissionRepository {
	return &UserServicePermissionRepository{db: db}
}

// Grant grants a user permission to access a service
func (r *UserServicePermissionRepository) Grant(userID int, serviceID string, grantedBy int, expiresAt *time.Time) error {
	query := `
		INSERT OR REPLACE INTO user_service_permissions (user_id, service_id, can_access, granted_by, granted_at, expires_at)
		VALUES (?, ?, TRUE, ?, ?, ?)
	`
	_, err := r.db.Exec(query, userID, serviceID, grantedBy, time.Now(), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to grant service permission: %w", err)
	}

	return nil
}

// Revoke revokes a user's permission to access a service
func (r *UserServicePermissionRepository) Revoke(userID int, serviceID string) error {
	query := `UPDATE user_service_permissions SET can_access = FALSE WHERE user_id = ? AND service_id = ?`
	_, err := r.db.Exec(query, userID, serviceID)
	if err != nil {
		return fmt.Errorf("failed to revoke service permission: %w", err)
	}

	return nil
}

// CheckPermission checks if a user has permission to access a service
func (r *UserServicePermissionRepository) CheckPermission(userID int, serviceID string) (bool, error) {
	var canAccess bool
	query := `
		SELECT can_access FROM user_service_permissions 
		WHERE user_id = ? AND service_id = ? AND (expires_at IS NULL OR expires_at > ?)
	`
	err := r.db.Get(&canAccess, query, userID, serviceID, time.Now())
	if err != nil {
		// If no explicit permission exists, check if service is public or user has sufficient role
		return false, nil
	}

	return canAccess, nil
}

// ListUserServices lists all services a user has access to
func (r *UserServicePermissionRepository) ListUserServices(userID int) ([]*RegisteredService, error) {
	query := `
		SELECT rs.id, rs.name, rs.display_name, rs.description, rs.service_url, rs.callback_url, rs.icon, rs.category, rs.is_public, rs.required_role, rs.status, rs.health_url, rs.last_healthy, rs.created_at, rs.updated_at
		FROM registered_services rs
		LEFT JOIN user_service_permissions usp ON rs.id = usp.service_id AND usp.user_id = ?
		WHERE rs.status = 'active' AND (
			rs.is_public = TRUE OR 
			(usp.can_access = TRUE AND (usp.expires_at IS NULL OR usp.expires_at > ?))
		)
		ORDER BY rs.category, rs.display_name
	`
	rows, err := r.db.Query(query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to list user services: %w", err)
	}
	defer rows.Close()

	var services []*RegisteredService
	for rows.Next() {
		var service RegisteredService
		err := rows.Scan(&service.ID, &service.Name, &service.DisplayName, &service.Description, &service.ServiceURL, &service.CallbackURL, &service.Icon, &service.Category, &service.IsPublic, &service.RequiredRole, &service.Status, &service.HealthURL, &service.LastHealthy, &service.CreatedAt, &service.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user service: %w", err)
		}
		services = append(services, &service)
	}

	return services, nil
}

// ServiceHealthCheckRepository provides database operations for service health checks
type ServiceHealthCheckRepository struct {
	db *DB
}

// NewServiceHealthCheckRepository creates a new service health check repository
func NewServiceHealthCheckRepository(db *DB) *ServiceHealthCheckRepository {
	return &ServiceHealthCheckRepository{db: db}
}

// Record records a health check result
func (r *ServiceHealthCheckRepository) Record(check *ServiceHealthCheck) error {
	query := `
		INSERT INTO service_health_checks (service_id, is_healthy, response_time, error_message, checked_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, check.ServiceID, check.IsHealthy, check.ResponseTime, check.ErrorMessage, check.CheckedAt)
	if err != nil {
		return fmt.Errorf("failed to record health check: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get health check ID: %w", err)
	}

	check.ID = int(id)
	return nil
}

// GetLatest gets the latest health check for a service
func (r *ServiceHealthCheckRepository) GetLatest(serviceID string) (*ServiceHealthCheck, error) {
	var check ServiceHealthCheck
	query := `
		SELECT id, service_id, is_healthy, response_time, error_message, checked_at
		FROM service_health_checks 
		WHERE service_id = ? 
		ORDER BY checked_at DESC 
		LIMIT 1
	`
	err := r.db.Get(&check, query, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest health check: %w", err)
	}

	return &check, nil
}

// GetHistory gets health check history for a service
func (r *ServiceHealthCheckRepository) GetHistory(serviceID string, limit int) ([]*ServiceHealthCheck, error) {
	query := `
		SELECT id, service_id, is_healthy, response_time, error_message, checked_at
		FROM service_health_checks 
		WHERE service_id = ? 
		ORDER BY checked_at DESC 
		LIMIT ?
	`
	rows, err := r.db.Query(query, serviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get health check history: %w", err)
	}
	defer rows.Close()

	var checks []*ServiceHealthCheck
	for rows.Next() {
		var check ServiceHealthCheck
		err := rows.Scan(&check.ID, &check.ServiceID, &check.IsHealthy, &check.ResponseTime, &check.ErrorMessage, &check.CheckedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health check: %w", err)
		}
		checks = append(checks, &check)
	}

	return checks, nil
}

// CleanupOldChecks removes old health check records
func (r *ServiceHealthCheckRepository) CleanupOldChecks(olderThan time.Time) error {
	query := `DELETE FROM service_health_checks WHERE checked_at < ?`
	_, err := r.db.Exec(query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to cleanup old health checks: %w", err)
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
