package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func jwtSecret() []byte {
	return []byte(envOr("JWT_SECRET", "intervoice-dev-secret"))
}

func parseAdminJWT(request *http.Request) (userID int64, role string, ok bool) {
	authHeader := request.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, "", false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil || token == nil || !token.Valid {
		return 0, "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", false
	}
	role, _ = claims["role"].(string)
	userID, ok = parseSubClaimAdmin(claims)
	if !ok {
		return 0, "", false
	}
	return userID, role, true
}

func parseSubClaimAdmin(claims jwt.MapClaims) (int64, bool) {
	raw, exists := claims["sub"]
	if !exists || raw == nil {
		return 0, false
	}
	switch value := raw.(type) {
	case float64:
		return int64(value), true
	case json.Number:
		parsed, err := value.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// requireAdminJWT writes 401/403 and returns false if the caller is not an admin.
func requireAdminJWT(writer http.ResponseWriter, request *http.Request) (actorUserID int64, ok bool) {
	userID, role, valid := parseAdminJWT(request)
	if !valid {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return 0, false
	}
	if role != "admin" {
		writeJSON(writer, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return 0, false
	}
	return userID, true
}
