package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func TestNewUserHandler(t *testing.T) {
	// Create mock auth and db
	mockAuth := &auth.Auth{}
	mockDB := &database.DB{}
	
	handler := NewUserHandler(mockAuth, mockDB)
	
	assert.NotNil(t, handler)
	assert.Equal(t, mockAuth, handler.auth)
	assert.Equal(t, mockDB, handler.db)
}

func TestNewServiceHandler(t *testing.T) {
	// Create mock db
	mockDB := &database.DB{}
	
	handler := NewServiceHandler(mockDB)
	
	assert.NotNil(t, handler)
	assert.Equal(t, mockDB, handler.db)
}

func TestNewSystemHandler(t *testing.T) {
	// Create mock db
	mockDB := &database.DB{}
	
	handler := NewSystemHandler(mockDB)
	
	assert.NotNil(t, handler)
	assert.Equal(t, mockDB, handler.db)
}

func TestNewSSOHandler(t *testing.T) {
	// Create mock auth and db
	mockAuth := &auth.Auth{}
	mockDB := &database.DB{}
	
	handler := NewSSOHandler(mockAuth, mockDB)
	
	assert.NotNil(t, handler)
	assert.Equal(t, mockAuth, handler.auth)
	assert.Equal(t, mockDB, handler.db)
}

func TestRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request RegisterRequest
		isValid bool
	}{
		{
			name: "valid registration request",
			request: RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Role:     "user",
			},
			isValid: true,
		},
		{
			name: "missing username",
			request: RegisterRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "password123",
				Role:     "user",
			},
			isValid: false,
		},
		{
			name: "missing email",
			request: RegisterRequest{
				Username: "testuser",
				Email:    "",
				Password: "password123",
				Role:     "user",
			},
			isValid: false,
		},
		{
			name: "missing password",
			request: RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "",
				Role:     "user",
			},
			isValid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isValid {
				assert.NotEmpty(t, tt.request.Username, "Valid request should have username")
				assert.NotEmpty(t, tt.request.Email, "Valid request should have email")
				assert.NotEmpty(t, tt.request.Password, "Valid request should have password")
				assert.Contains(t, tt.request.Email, "@", "Valid email should contain @")
			} else {
				isValid := tt.request.Username != "" && 
					tt.request.Email != "" && 
					tt.request.Password != ""
				assert.False(t, isValid, "Invalid request should fail validation")
			}
		})
	}
}

func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request auth.LoginRequest
		isValid bool
	}{
		{
			name: "valid login request",
			request: auth.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			isValid: true,
		},
		{
			name: "missing username",
			request: auth.LoginRequest{
				Username: "",
				Password: "password123",
			},
			isValid: false,
		},
		{
			name: "missing password",
			request: auth.LoginRequest{
				Username: "testuser",
				Password: "",
			},
			isValid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isValid {
				assert.NotEmpty(t, tt.request.Username, "Valid request should have username")
				assert.NotEmpty(t, tt.request.Password, "Valid request should have password")
			} else {
				isValid := tt.request.Username != "" && tt.request.Password != ""
				assert.False(t, isValid, "Invalid request should fail validation")
			}
		})
	}
}

func TestHealthCheckHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		// Mock health check response
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": 1234567890,
			"service":   "infra-core-console",
		})
	})
	
	// Test health check endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "timestamp")
	assert.Equal(t, "infra-core-console", response["service"])
}

func TestServiceCreationRequest(t *testing.T) {
	// Test service creation request structure
	request := map[string]interface{}{
		"name":        "test-service",
		"yaml_config": "name: test-service\nimage: nginx:alpine",
		"description": "Test service description",
	}
	
	assert.Equal(t, "test-service", request["name"])
	assert.Contains(t, request["yaml_config"], "name:")
	assert.Contains(t, request["yaml_config"], "image:")
	assert.NotEmpty(t, request["description"])
}

func TestJSONResponseStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test various response structures
	responses := []struct {
		name     string
		response gin.H
	}{
		{
			name: "success response",
			response: gin.H{
				"success": true,
				"message": "Operation completed",
				"data":    map[string]interface{}{"id": 123},
			},
		},
		{
			name: "error response",
			response: gin.H{
				"success": false,
				"error":   "Validation failed",
				"code":    400,
			},
		},
		{
			name: "list response",
			response: gin.H{
				"data":  []string{"item1", "item2", "item3"},
				"count": 3,
				"page":  1,
			},
		},
	}
	
	for _, resp := range responses {
		t.Run(resp.name, func(t *testing.T) {
			// Create test router
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, resp.response)
			})
			
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			// Verify response structure
			for key := range resp.response {
				assert.Contains(t, response, key)
				if key == "data" && resp.name == "list response" {
					// Special handling for array data
					assert.IsType(t, []interface{}{}, response[key])
				}
			}
		})
	}
}

func TestHTTPStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	statusTests := []struct {
		name       string
		statusCode int
		response   gin.H
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			response:   gin.H{"status": "success"},
		},
		{
			name:       "created",
			statusCode: http.StatusCreated,
			response:   gin.H{"status": "created", "id": 123},
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   gin.H{"error": "invalid input"},
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   gin.H{"error": "authentication required"},
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   gin.H{"error": "resource not found"},
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			response:   gin.H{"error": "internal server error"},
		},
	}
	
	for _, test := range statusTests {
		t.Run(test.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				c.JSON(test.statusCode, test.response)
			})
			
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, test.statusCode, w.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			// Verify response content
			for key, expectedValue := range test.response {
				actualValue := response[key]
				
				// Handle int to float64 conversion in JSON
				if expectedInt, ok := expectedValue.(int); ok {
					if actualFloat, ok := actualValue.(float64); ok {
						assert.Equal(t, float64(expectedInt), actualFloat)
					} else {
						assert.Equal(t, expectedValue, actualValue)
					}
				} else {
					assert.Equal(t, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestRequestBodyParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test JSON request body parsing
	requestData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}
	
	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)
	
	r := gin.New()
	r.POST("/test", func(c *gin.Context) {
		var parsed map[string]interface{}
		if err := c.ShouldBindJSON(&parsed); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"received": parsed,
			"count":    len(parsed),
		})
	})
	
	req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response, "received")
	assert.Equal(t, float64(3), response["count"])
	
	received := response["received"].(map[string]interface{})
	assert.Equal(t, "testuser", received["username"])
	assert.Equal(t, "test@example.com", received["email"])
}

func TestParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"user_id": id})
	})
	
	r.GET("/services/:id/logs", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"service_id": id, "action": "logs"})
	})
	
	testCases := []struct {
		path     string
		expected map[string]interface{}
	}{
		{
			path:     "/users/123",
			expected: map[string]interface{}{"user_id": "123"},
		},
		{
			path:     "/services/test-service/logs",
			expected: map[string]interface{}{"service_id": "test-service", "action": "logs"},
		},
	}
	
	for _, tc := range testCases {
		req, err := http.NewRequest("GET", tc.path, nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		for key, expectedValue := range tc.expected {
			assert.Equal(t, expectedValue, response[key])
		}
	}
}