package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

// AuthMiddleware creates authentication middleware with session support
func AuthMiddleware(authService *auth.Auth, db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Check session if exists
		if claims.SessionID != "" && db != nil {
			sessionRepo := db.SSOSessionRepository()
			session, err := sessionRepo.GetByTokenHash(authService.HashSessionToken(token))
			if err != nil || !session.IsActive {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
				c.Abort()
				return
			}

			// Update last used timestamp
			sessionRepo.UpdateLastUsed(session.ID)
		}

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("session_id", claims.SessionID)
		c.Set("permissions", claims.Permissions)
		c.Set("services", claims.Services)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireRole creates role-based authorization middleware
func RequireRole(authService *auth.Auth, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		if !authService.RequireRole(userRole.(string), requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SSOAuthMiddleware creates SSO authentication middleware
func SSOAuthMiddleware(authService *auth.Auth, db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from various sources
		token := extractToken(c)
		if token == "" {
			// No token provided, redirect to SSO login
			redirectToSSOLogin(c)
			return
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			// Invalid token, redirect to SSO login
			redirectToSSOLogin(c)
			return
		}

		// Check if session is still valid
		if claims.SessionID != "" {
			sessionRepo := db.SSOSessionRepository()
			session, err := sessionRepo.GetByTokenHash(authService.HashSessionToken(token))
			if err != nil || !session.IsActive {
				// Session expired or invalid, redirect to SSO login
				redirectToSSOLogin(c)
				return
			}

			// Update last used timestamp
			sessionRepo.UpdateLastUsed(session.ID)
		}

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("session_id", claims.SessionID)
		c.Set("permissions", claims.Permissions)
		c.Set("services", claims.Services)
		c.Set("claims", claims)

		c.Next()
	}
}

// SSOServiceAuthMiddleware creates service-specific SSO authentication middleware
func SSOServiceAuthMiddleware(authService *auth.Auth, db *database.DB, serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user has access to this specific service
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		userClaims := claims.(*auth.Claims)
		
		// Check service access
		if !authService.HasServiceAccess(userClaims, serviceName) {
			// Check in database for explicit permissions
			permRepo := db.UserServicePermissionRepository()
			serviceRepo := db.RegisteredServiceRepository()
			
			service, err := serviceRepo.GetByName(serviceName)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
				c.Abort()
				return
			}

			// Check role requirements
			if !authService.RequireRole(userClaims.Role, service.RequiredRole) && !service.IsPublic {
				hasPermission, err := permRepo.CheckPermission(userClaims.UserID, service.ID)
				if err != nil || !hasPermission {
					c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this service"})
					c.Abort()
					return
				}
			}
		}

		c.Set("service_name", serviceName)
		c.Next()
	}
}

// SSOTokenValidationMiddleware validates SSO tokens specifically
func SSOTokenValidationMiddleware(authService *auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("sso_token")
		if token == "" {
			token = c.GetHeader("X-SSO-Token")
		}

		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SSO token required"})
			c.Abort()
			return
		}

		claims, err := authService.ValidateSSOToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid SSO token"})
			c.Abort()
			return
		}

		// Add SSO claims to context
		c.Set("sso_claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("source_service", claims.SourceService)
		c.Set("target_service", claims.TargetService)

		c.Next()
	}
}

// RequireServicePermission creates middleware that requires specific service permission
func RequireServicePermission(authService *auth.Auth, db *database.DB, serviceID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		userRole, _ := c.Get("role")

		// Admin users have access to all services
		if userRole == "admin" {
			c.Next()
			return
		}

		// Check service permission
		permRepo := db.UserServicePermissionRepository()
		hasPermission, err := permRepo.CheckPermission(userID.(int), serviceID)
		if err != nil || !hasPermission {
			// Check if service is public
			serviceRepo := db.RegisteredServiceRepository()
			service, err := serviceRepo.GetByID(serviceID)
			if err != nil || !service.IsPublic {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this service"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// extractToken extracts authentication token from various sources
func extractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
			return tokenParts[1]
		}
	}

	// Try query parameter
	if token := c.Query("token"); token != "" {
		return token
	}

	// Try SSO token parameter
	if token := c.Query("sso_token"); token != "" {
		return token
	}

	// Try cookie
	if cookie, err := c.Cookie("auth_token"); err == nil {
		return cookie
	}

	return ""
}

// redirectToSSOLogin redirects the user to SSO login with the current URL as redirect target
func redirectToSSOLogin(c *gin.Context) {
	// Build current URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	
	currentURL := fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.Request.RequestURI)
	redirectURL := url.QueryEscape(currentURL)
	
	// Redirect to SSO login page
	ssoLoginURL := fmt.Sprintf("/login?redirect=%s", redirectURL)
	
	// For API requests, return JSON instead of redirect
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"login_url": ssoLoginURL,
		})
		c.Abort()
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, ssoLoginURL)
	c.Abort()
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// RecoveryMiddleware handles panics
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}
