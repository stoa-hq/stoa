package auth

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	minSecretLength = 32
	defaultSecret   = "change-me-in-production"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID `json:"uid"`
	Email    string    `json:"email"`
	UserType string    `json:"utype"` // "admin", "customer"
	Role     string    `json:"role"`
	Type     TokenType `json:"type"`
}

type JWTManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) (*JWTManager, error) {
	if secret == defaultSecret {
		return nil, errors.New("jwt: default secret 'change-me-in-production' must not be used — set auth.jwt_secret in config")
	}
	if len(secret) < minSecretLength {
		return nil, fmt.Errorf("jwt: secret must be at least %d bytes, got %d", minSecretLength, len(secret))
	}
	if len(secret) < 64 {
		log.Println("WARNING: jwt secret is shorter than 64 bytes — consider using a stronger secret")
	}
	return &JWTManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}, nil
}

func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email, userType, role string) (string, error) {
	return m.generateToken(userID, email, userType, role, AccessToken, m.accessTokenTTL)
}

func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, email, userType, role string) (string, error) {
	return m.generateToken(userID, email, userType, role, RefreshToken, m.refreshTokenTTL)
}

func (m *JWTManager) generateToken(userID uuid.UUID, email, userType, role string, tokenType TokenType, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:   userID,
		Email:    email,
		UserType: userType,
		Role:     role,
		Type:     tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
