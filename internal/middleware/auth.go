package middleware

import (
	"fmt"
	"strings"

	apierrors "github.com/Kupfy/feeds-crawler/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// UserClaims represents the JWT claims structure
type UserClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

//type AuthMiddleware interface {
//	RequireAuth() gin.HandlerFunc
//	RequireRole() gin.HandlerFunc
//	OptionalAuth() gin.HandlerFunc
//}

// AuthMiddleware creates a JWT authentication middleware
type AuthMiddleware struct {
	jwtSecret string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret string) AuthMiddleware {
	return AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// RequireAuth validates JWT token and sets user context
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			err := apierrors.NewUnauthorizedError("Authorization header required")
			c.JSON(err.Status, gin.H{"code": err.Code, "message": err.Message})
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			err := apierrors.NewUnauthorizedError("Invalid authorization header format")
			c.JSON(err.Status, gin.H{"code": err.Code, "message": err.Message})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(am.jwtSecret), nil
		})

		if err != nil {
			apiErr := apierrors.NewUnauthorizedError("Invalid or expired token")
			c.JSON(apiErr.Status, gin.H{"code": apiErr.Code, "message": apiErr.Message})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*UserClaims)
		if !ok || !token.Valid {
			apiErr := apierrors.NewUnauthorizedError("Invalid token claims")
			c.JSON(apiErr.Status, gin.H{"code": apiErr.Code, "message": apiErr.Message})
			c.Abort()
			return
		}

		// Set user information in context for handlers to use
		c.Set("userID", claims.UserID)
		c.Set("userRoles", claims.Roles)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireRole validates that the authenticated user has the specified role
func (am *AuthMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// User should already be authenticated by RequireAuth middleware
		roles, exists := c.Get("userRoles")
		if !exists {
			apiErr := apierrors.NewForbiddenError("User roles not found in context")
			c.JSON(apiErr.Status, gin.H{"code": apiErr.Code, "message": apiErr.Message})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			apiErr := apierrors.NewInternalError("Invalid roles format")
			c.JSON(apiErr.Status, gin.H{"code": apiErr.Code, "message": apiErr.Message})
			c.Abort()
			return
		}

		// Check if user has required role
		hasRole := false
		for _, r := range userRoles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			apiErr := apierrors.NewForbiddenError(fmt.Sprintf("Role '%s' required", role))
			c.JSON(apiErr.Status, gin.H{"code": apiErr.Code, "message": apiErr.Message})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth validates JWT if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		// If token is provided, validate it
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]

			token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(am.jwtSecret), nil
			})

			if err == nil {
				if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
					c.Set("userID", claims.UserID)
					c.Set("userRoles", claims.Roles)
					c.Set("claims", claims)
				}
			}
		}

		c.Next()
	}
}
