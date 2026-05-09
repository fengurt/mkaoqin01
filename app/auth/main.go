package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"intervoice/dbschema"
)

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string         `json:"token"`
	User  map[string]any `json:"user"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type adminResetPasswordRequest struct {
	UserID      int64  `json:"userId"`
	NewPassword string `json:"newPassword"`
}

func main() {
	database := mustOpenDatabase()
	defer database.Close()
	mustInitDatabase(database)

	addr := envOr("AUTH_ADDR", ":8001")
	hsrv := khttp.NewServer(khttp.Address(addr))

	hsrv.HandleFunc("/v1/auth/login", func(writer http.ResponseWriter, request *http.Request) {
		var payload loginRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		row := database.QueryRow("SELECT id, role, display_name FROM users WHERE account = ? AND password = ?", payload.Account, payload.Password)
		var userID int64
		var role string
		var displayName string
		if err := row.Scan(&userID, &role, &displayName); err != nil {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "account or password invalid"})
			return
		}

		tokenString, err := signJWT(userID, role)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
			return
		}

		writeJSON(writer, http.StatusOK, loginResponse{
			Token: tokenString,
			User: map[string]any{"id": userID, "role": role, "displayName": displayName},
		})
	})

	hsrv.HandleFunc("/v1/auth/wechat", func(writer http.ResponseWriter, request *http.Request) {
		tokenString, _ := signJWT(1, "employee")
		writeJSON(writer, http.StatusOK, loginResponse{
			Token: tokenString,
			User:  map[string]any{"id": 1, "role": "employee", "displayName": "微信员工演示账号"},
		})
	})

	hsrv.HandleFunc("/v1/auth/users", func(writer http.ResponseWriter, request *http.Request) {
		userID, role, ok := parseJWTFromRequest(request)
		if !ok || userID == 0 {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if role != "admin" {
			writeJSON(writer, http.StatusForbidden, map[string]string{"error": "admin required"})
			return
		}

		rows, err := database.Query("SELECT id, account, role, display_name FROM users ORDER BY id ASC")
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query users failed"})
			return
		}
		defer rows.Close()

		items := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var account string
			var itemRole string
			var displayName string
			_ = rows.Scan(&id, &account, &itemRole, &displayName)
			items = append(items, map[string]any{
				"id":          id,
				"account":     account,
				"role":        itemRole,
				"displayName": displayName,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"items": items})
	})

	hsrv.HandleFunc("/v1/auth/password/change", func(writer http.ResponseWriter, request *http.Request) {
		userID, _, ok := parseJWTFromRequest(request)
		if !ok || userID == 0 {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		var payload passwordChangeRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if len(payload.NewPassword) < 6 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "new password too short"})
			return
		}

		var existingPassword string
		if err := database.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&existingPassword); err != nil {
			writeJSON(writer, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		if existingPassword != payload.CurrentPassword {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "current password incorrect"})
			return
		}
		if _, err := database.Exec("UPDATE users SET password = ? WHERE id = ?", payload.NewPassword, userID); err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "update password failed"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	hsrv.HandleFunc("/v1/auth/password/reset", func(writer http.ResponseWriter, request *http.Request) {
		userID, role, ok := parseJWTFromRequest(request)
		if !ok || userID == 0 {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if role != "admin" {
			writeJSON(writer, http.StatusForbidden, map[string]string{"error": "admin required"})
			return
		}

		var payload adminResetPasswordRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if payload.UserID == 0 || len(payload.NewPassword) < 6 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
			return
		}
		if _, err := database.Exec("UPDATE users SET password = ? WHERE id = ?", payload.NewPassword, payload.UserID); err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "reset password failed"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	hsrv.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"service": "auth-svc", "status": "ok"})
	})

	app := kratos.New(kratos.Name("auth-svc"), kratos.Server(hsrv))
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func signJWT(userID int64, role string) (string, error) {
	secret := []byte(envOr("JWT_SECRET", "intervoice-dev-secret"))
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(12 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func parseSubClaim(claims jwt.MapClaims) (int64, bool) {
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

func parseJWTFromRequest(request *http.Request) (int64, string, bool) {
	authHeader := request.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, "", false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	secret := []byte(envOr("JWT_SECRET", "intervoice-dev-secret"))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing method")
		}
		return secret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return 0, "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", false
	}
	role, _ := claims["role"].(string)
	userID, subOK := parseSubClaim(claims)
	if !subOK {
		return 0, "", false
	}
	return userID, role, true
}

func mustOpenDatabase() *sql.DB {
	dbPath := envOr("DB_PATH", "../../data/intervoice.db")
	database, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		panic(err)
	}
	return database
}

func mustInitDatabase(database *sql.DB) {
	if err := dbschema.ApplySQLite(database); err != nil {
		panic(err)
	}
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
