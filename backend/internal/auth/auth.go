package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Claims struct {
	Sub      int64  `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Type     string `json:"type"`
}

func CreateAccessToken(userID int64, username, secret string, ttl time.Duration) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	claims, _ := json.Marshal(Claims{Sub: userID, Username: username, Exp: time.Now().Add(ttl).Unix(), Type: "access"})
	unsigned := encode(header) + "." + encode(claims)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + encode(mac.Sum(nil)), nil
}

func ParseAccessToken(token, secret string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return Claims{}, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if json.Unmarshal(payload, &claims) != nil || claims.Type != "access" || claims.Sub < 1 || time.Now().Unix() >= claims.Exp {
		return Claims{}, errors.New("expired or invalid token")
	}
	return claims, nil
}

func NewRefreshToken() (plain, hash string, err error) {
	bytes := make([]byte, 48)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(bytes)
	return plain, HashRefreshToken(plain), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func encode(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }
func Bearer(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid authorization header: %s", strconv.Quote(header))
	}
	return parts[1], nil
}
