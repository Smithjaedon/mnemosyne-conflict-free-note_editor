package middleware

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const ACCESS_TOKEN_EXPIRE_MINUTES = 15

var secretKey []byte

func init() {
	key := os.Getenv("SECRET_KEY")
	if key == "" {
		log.Fatal("SECRET_KEY environment variable is not set")
	}
	secretKey = []byte(key)
}

func CreateAccessToken(userID string) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * ACCESS_TOKEN_EXPIRE_MINUTES).Unix(),
	}).SignedString([]byte(secretKey))
	return token, err
}

func ValidateAccessToken(tokenStr string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return &claims, nil
}

func GetUserIDFromClaims(claims *jwt.MapClaims) (string, error) {
	sub, ok := (*claims)["sub"].(string)
	if !ok {
		return "", errors.New("invalid token subject")
	}
	return sub, nil
}

func RequireAuth(tokenStr string) (string, error) {
	claims, err := ValidateAccessToken(tokenStr)
	if err != nil {
		return "", err
	}
	return GetUserIDFromClaims(claims)
}
