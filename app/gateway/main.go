package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	addr := envOr("GATEWAY_ADDR", ":8010")
	authBaseURL := envOr("AUTH_BASE_URL", "http://127.0.0.1:8001")
	attendanceBaseURL := envOr("ATTENDANCE_BASE_URL", "http://127.0.0.1:8002")
	voiceBaseURL := envOr("VOICE_BASE_URL", "http://127.0.0.1:8003")
	adminBaseURL := envOr("ADMIN_BASE_URL", "http://127.0.0.1:8004")
	hsrv := khttp.NewServer(khttp.Address(addr))

	hsrv.HandleFunc("/", handleOptions)
	hsrv.HandleFunc("/v1/auth/login", withCORS(proxyJSON(authBaseURL+"/v1/auth/login", false)))
	hsrv.HandleFunc("/v1/auth/wechat", withCORS(proxyJSON(authBaseURL+"/v1/auth/wechat", false)))
	hsrv.HandleFunc("/v1/auth/users", withCORS(proxyGET(authBaseURL+"/v1/auth/users", true)))
	hsrv.HandleFunc("/v1/auth/password/change", withCORS(proxyJSON(authBaseURL+"/v1/auth/password/change", true)))
	hsrv.HandleFunc("/v1/auth/password/reset", withCORS(proxyJSON(authBaseURL+"/v1/auth/password/reset", true)))
	hsrv.HandleFunc("/v1/voice/recognize", withCORS(proxyMultipart(voiceBaseURL+"/v1/voice/recognize", true)))
	hsrv.HandleFunc("/v1/attendance/submit", withCORS(proxyJSON(attendanceBaseURL+"/v1/attendance/submit", true)))
	hsrv.HandleFunc("/v1/attendance/today", withCORS(proxyGET(attendanceBaseURL+"/v1/attendance/today", true)))
	hsrv.HandleFunc("/v1/attendance/summary", withCORS(proxyGET(attendanceBaseURL+"/v1/attendance/summary", true)))
	hsrv.HandleFunc("/v1/attendance/list", withCORS(proxyGET(attendanceBaseURL+"/v1/attendance/list", true)))
	hsrv.HandleFunc("/v1/attendance/by-date", withCORS(proxyGET(attendanceBaseURL+"/v1/attendance/by-date", true)))
	hsrv.HandleFunc("/v1/rewards/me", withCORS(proxyGET(attendanceBaseURL+"/v1/rewards/me", true)))
	hsrv.HandleFunc("/v1/rewards/ack", withCORS(proxyJSON(attendanceBaseURL+"/v1/rewards/ack", true)))
	hsrv.HandleFunc("/v1/leads/feed", withCORS(proxyGET(attendanceBaseURL+"/v1/leads/feed", true)))
	hsrv.HandleFunc("/v1/leads/detail", withCORS(proxyGET(attendanceBaseURL+"/v1/leads/detail", true)))
	hsrv.HandleFunc("/v1/leads/pick-up", withCORS(proxyJSON(attendanceBaseURL+"/v1/leads/pick-up", true)))
	hsrv.HandleFunc("/v1/leads/follow-up", withCORS(proxyJSON(attendanceBaseURL+"/v1/leads/follow-up", true)))
	hsrv.HandleFunc("/v1/catalog/locations", withCORS(proxyGET(attendanceBaseURL+"/v1/catalog/locations", true)))
	hsrv.HandleFunc("/v1/catalog/schedule-quick", withCORS(proxyGET(attendanceBaseURL+"/v1/catalog/schedule-quick", false)))
	// Catalog-only lists (shift_types / activity_types); same data as admin UI. Saving still requires JWT via POST /v1/schedule/day.
	hsrv.HandleFunc("/v1/schedule/day-options", withCORS(proxyGET(attendanceBaseURL+"/v1/schedule/day-options", false)))
	hsrv.HandleFunc("/v1/schedule/day", withCORS(proxyScheduleDay(attendanceBaseURL)))
	hsrv.HandleFunc("/v1/schedule/month", withCORS(proxyGET(attendanceBaseURL+"/v1/schedule/month", true)))
	hsrv.HandleFunc("/v1/fortune/day", withCORS(proxyGET(attendanceBaseURL+"/v1/fortune/day", true)))
	hsrv.HandleFunc("/v1/fortune/month", withCORS(proxyGET(attendanceBaseURL+"/v1/fortune/month", true)))
	hsrv.HandleFunc("/uploads/fortune/", withCORS(proxyUploadsFortune(attendanceBaseURL)))
	hsrv.HandleFunc("/v1/admin/fortune/month", withCORS(proxyGET(adminBaseURL+"/v1/admin/fortune/month", true)))
	hsrv.HandleFunc("/v1/admin/fortune/upload", withCORS(proxyMultipartForward(adminBaseURL+"/v1/admin/fortune/upload", true)))
	hsrv.HandleFunc("/v1/admin/fortune/assign", withCORS(proxyJSON(adminBaseURL+"/v1/admin/fortune/assign", true)))
	hsrv.HandleFunc("/v1/admin/fortune/sync", withCORS(proxyJSON(adminBaseURL+"/v1/admin/fortune/sync", true)))
	hsrv.HandleFunc("/v1/admin/schedule/grid/export", withCORS(proxyGET(adminBaseURL+"/v1/admin/schedule/grid/export", true)))
	hsrv.HandleFunc("/v1/admin/schedule/grid/import", withCORS(proxyJSON(adminBaseURL+"/v1/admin/schedule/grid/import", true)))
	hsrv.HandleFunc("/v1/admin/board", withCORS(proxyGET(adminBaseURL+"/v1/admin/board", true)))
	hsrv.HandleFunc("/v1/admin/summary", withCORS(proxyGET(adminBaseURL+"/v1/admin/summary", true)))
	hsrv.HandleFunc("/v1/admin/report", withCORS(proxyGET(adminBaseURL+"/v1/admin/report", true)))
	hsrv.HandleFunc("/v1/admin/team", withCORS(proxyGET(adminBaseURL+"/v1/admin/team", true)))
	hsrv.HandleFunc("/v1/admin/leads", withCORS(proxyGET(adminBaseURL+"/v1/admin/leads", true)))
	hsrv.HandleFunc("/v1/admin/data/meta", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/meta", true)))
	hsrv.HandleFunc("/v1/admin/data/schema", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/schema", true)))
	hsrv.HandleFunc("/v1/admin/data/users", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/users", true)))
	hsrv.HandleFunc("/v1/admin/data/attendance", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/attendance", true)))
	hsrv.HandleFunc("/v1/admin/data/attendance/upsert", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/attendance/upsert", true)))
	hsrv.HandleFunc("/v1/admin/data/users/patch", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/users/patch", true)))
	hsrv.HandleFunc("/v1/admin/data/location-catalog", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/location-catalog", true)))
	hsrv.HandleFunc("/v1/admin/data/location-catalog/upsert", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/location-catalog/upsert", true)))
	hsrv.HandleFunc("/v1/admin/data/location-catalog/delete", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/location-catalog/delete", true)))
	hsrv.HandleFunc("/v1/admin/data/schedule-quick-sections", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/schedule-quick-sections", true)))
	hsrv.HandleFunc("/v1/admin/data/schedule-quick-sections/upsert", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/schedule-quick-sections/upsert", true)))
	hsrv.HandleFunc("/v1/admin/data/schedule-quick-sections/delete", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/schedule-quick-sections/delete", true)))
	hsrv.HandleFunc("/v1/admin/data/activity-types", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/activity-types", true)))
	hsrv.HandleFunc("/v1/admin/data/activity-types/upsert", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/activity-types/upsert", true)))
	hsrv.HandleFunc("/v1/admin/data/shift-types", withCORS(proxyGET(adminBaseURL+"/v1/admin/data/shift-types", true)))
	hsrv.HandleFunc("/v1/admin/data/shift-types/upsert", withCORS(proxyJSON(adminBaseURL+"/v1/admin/data/shift-types/upsert", true)))
	hsrv.HandleFunc("/healthz", withCORS(func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"service": "gateway-svc", "status": "ok"})
	}))

	app := kratos.New(kratos.Name("gateway-svc"), kratos.Server(hsrv))
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func proxyJSON(targetURL string, authRequired bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authRequired && !validateJWT(request) {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		bodyBytes, _ := io.ReadAll(request.Body)
		targetRequest, _ := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
		targetRequest.Header.Set("Content-Type", "application/json")
		if authRequired {
			targetRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
		}
		proxyRequest(writer, targetRequest)
	}
}

func proxyScheduleDay(baseURL string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !validateJWT(request) {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		target := baseURL + "/v1/schedule/day"
		switch request.Method {
		case http.MethodGet:
			if request.URL.RawQuery != "" {
				target = target + "?" + request.URL.RawQuery
			}
			targetRequest, _ := http.NewRequest(http.MethodGet, target, nil)
			targetRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
			proxyRequest(writer, targetRequest)
		case http.MethodPost:
			bodyBytes, _ := io.ReadAll(request.Body)
			targetRequest, _ := http.NewRequest(http.MethodPost, target, bytes.NewReader(bodyBytes))
			targetRequest.Header.Set("Content-Type", "application/json")
			targetRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
			proxyRequest(writer, targetRequest)
		default:
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}
}

func proxyGET(targetURL string, authRequired bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authRequired && !validateJWT(request) {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		queryString := request.URL.RawQuery
		requestURL := targetURL
		if queryString != "" {
			requestURL = requestURL + "?" + queryString
		}
		targetRequest, _ := http.NewRequest(http.MethodGet, requestURL, nil)
		if authRequired {
			targetRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
		}
		proxyRequest(writer, targetRequest)
	}
}

func proxyUploadsFortune(attendanceBaseURL string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		target := attendanceBaseURL + request.URL.Path
		if request.URL.RawQuery != "" {
			target = target + "?" + request.URL.RawQuery
		}
		targetRequest, _ := http.NewRequest(request.Method, target, request.Body)
		targetRequest.Header = request.Header.Clone()
		proxyRequest(writer, targetRequest)
	}
}

func proxyMultipartForward(targetURL string, authRequired bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authRequired && !validateJWT(request) {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err := request.ParseMultipartForm(16 << 20); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid multipart request"})
			return
		}
		var body bytes.Buffer
		writerForm := multipart.NewWriter(&body)
		for key, vals := range request.MultipartForm.Value {
			if len(vals) > 0 {
				_ = writerForm.WriteField(key, vals[0])
			}
		}
		for field, headers := range request.MultipartForm.File {
			for _, hdr := range headers {
				file, _ := hdr.Open()
				part, _ := writerForm.CreateFormFile(field, hdr.Filename)
				_, _ = io.Copy(part, file)
				_ = file.Close()
			}
		}
		writerForm.Close()
		targetRequest, _ := http.NewRequest(http.MethodPost, targetURL, &body)
		targetRequest.Header.Set("Content-Type", writerForm.FormDataContentType())
		if authRequired {
			targetRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
		}
		proxyRequest(writer, targetRequest)
	}
}

func proxyMultipart(targetURL string, authRequired bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authRequired && !validateJWT(request) {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		if err := request.ParseMultipartForm(16 << 20); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid multipart request"})
			return
		}
		files := request.MultipartForm.File["audio"]
		if len(files) == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "missing audio file"})
			return
		}
		file, _ := files[0].Open()
		defer file.Close()
		fileBytes, _ := io.ReadAll(file)

		var body bytes.Buffer
		writerForm := multipart.NewWriter(&body)
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "audio", files[0].Filename))
		originalContentType := files[0].Header.Get("Content-Type")
		if originalContentType == "" {
			originalContentType = "application/octet-stream"
		}
		partHeader.Set("Content-Type", originalContentType)
		part, _ := writerForm.CreatePart(partHeader)
		_, _ = part.Write(fileBytes)
		writerForm.Close()

		targetRequest, _ := http.NewRequest(http.MethodPost, targetURL, &body)
		targetRequest.Header.Set("Content-Type", writerForm.FormDataContentType())
		if authRequired {
			targetRequest.Header.Set("Authorization", request.Header.Get("Authorization"))
		}
		proxyRequest(writer, targetRequest)
	}
}

func proxyRequest(writer http.ResponseWriter, request *http.Request) {
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		writeJSON(writer, http.StatusBadGateway, map[string]string{"error": "upstream unavailable"})
		return
	}
	defer response.Body.Close()

	writer.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	writer.WriteHeader(response.StatusCode)
	_, _ = io.Copy(writer, response.Body)
}

func validateJWT(request *http.Request) bool {
	authHeader := request.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	secret := []byte(envOr("JWT_SECRET", "intervoice-dev-secret"))
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing method")
		}
		return secret, nil
	})
	return err == nil
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next(writer, request)
	}
}

func handleOptions(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
	writer.WriteHeader(http.StatusNoContent)
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
