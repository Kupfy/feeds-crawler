package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func main() {
	// Get JWT secret from environment or use default
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-key"
		fmt.Println("Using default JWT_SECRET: dev-secret-key")
	}

	// Parse command line arguments for user ID and roles
	userID := "test-user-123"
	roles := []string{"user"}

	if len(os.Args) > 1 {
		userID = os.Args[1]
	}
	if len(os.Args) > 2 {
		roles = os.Args[2:]
	}

	// Create claims
	claims := UserClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	fmt.Println("\n=== JWT Token Generated ===")
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Roles: %v\n", roles)
	fmt.Printf("Expires: %s\n", claims.ExpiresAt.Time.Format(time.RFC3339))
	fmt.Println("\nToken:")
	fmt.Println(tokenString)
}
