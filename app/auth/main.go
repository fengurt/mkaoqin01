package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
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
	subNumber, subOK := claims["sub"].(float64)
	if !subOK {
		return 0, "", false
	}
	return int64(subNumber), role, true
}

func mustOpenDatabase() *sql.DB {
	dbPath := envOr("DB_PATH", "../../data/intervoice.db")
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	return database
}

func mustInitDatabase(database *sql.DB) {
	schema := `CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  account TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  role TEXT NOT NULL,
  display_name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS attendance_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  location TEXT,
  reason TEXT,
  occurred_at TEXT NOT NULL,
  attachment_url TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS activity_types (
  code TEXT PRIMARY KEY,
  full_name TEXT NOT NULL,
  description TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS shift_types (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  duration_minutes INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS employee_schedules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  staff_id TEXT NOT NULL,
  staff_name TEXT NOT NULL,
  team_name TEXT NOT NULL,
  week_range TEXT NOT NULL,
  schedule_text TEXT NOT NULL,
  UNIQUE(staff_id, team_name, week_range)
);
INSERT OR IGNORE INTO users (id, account, password, role, display_name) VALUES
  (900001, 'admin', '123456a', 'admin', '系统管理员'),
  (900002, 'admin01', '123456a', 'admin', '系统管理员-兼容账号'),
  (900101, 'staff01', '123456a', 'employee', '员工演示账号'),
  (118919, '118919', '123456a', 'employee', 'Justin Lu'),
  (132369, '132369', '123456a', 'employee', 'Albee Liu'),
  (132387, '132387', '123456a', 'employee', 'Betty Zhang'),
  (132920, '132920', '123456a', 'employee', 'Heather Zou'),
  (135320, '135320', '123456a', 'employee', 'Simon Kok'),
  (111071, '111071', '123456a', 'employee', 'Sonia Song'),
  (142035, '142035', '123456a', 'employee', 'Ashley Lei'),
  (132922, '132922', '123456a', 'employee', 'Isaac Su'),
  (128943, '128943', '123456a', 'employee', 'Kalei Kong'),
  (134846, '134846', '123456a', 'employee', 'Simon Wu'),
  (137291, '137291', '123456a', 'employee', 'Emily Li'),
  (137420, '137420', '123456a', 'employee', 'Max Wang'),
  (139407, '139407', '123456a', 'employee', 'Stacey Pong'),
  (140957, '140957', '123456a', 'employee', 'Owen Liang'),
  (132921, '132921', '123456a', 'employee', 'Bella Guo'),
  (132386, '132386', '123456a', 'employee', 'Elva Ao'),
  (132923, '132923', '123456a', 'employee', 'Sky Wang'),
  (132924, '132924', '123456a', 'employee', 'William Chen'),
  (139436, '139436', '123456a', 'employee', 'Joyce Yi'),
  (141898, '141898', '123456a', 'employee', 'Leah Zhou'),
  (142133, '142133', '123456a', 'employee', 'Jeremy Cai'),
  (132945, '132945', '123456a', 'employee', 'Vicky Yue'),
  (140672, '140672', '123456a', 'employee', 'SiSi Sou'),
  (141780, '141780', '123456a', 'employee', 'Sammi Xian'),
  (141906, '141906', '123456a', 'employee', 'Duke Sui');
INSERT OR IGNORE INTO activity_types (code, full_name, description) VALUES
  ('RDO','Rest Day Off','常规休息日'),
  ('AL','Annual Leave','带薪年假'),
  ('PH','Public Holiday','公共假期'),
  ('PHCL','Public Holiday Compensatory Leave','公共假期加班补休'),
  ('RDOC','Rest Day Off Compensatory','休息日加班补休'),
  ('RV','Rest Vacation','其他类型带薪休假');
INSERT OR IGNORE INTO shift_types (code, name, start_time, end_time, duration_minutes) VALUES
  ('EARLY','早班','09:30','19:06',576),
  ('EARLY_MID','早中班','10:30','20:06',576),
  ('MID','中班','11:00','20:36',576),
  ('MID_LATE','中晚班','12:00','21:36',576),
  ('LATE','晚班','13:30','23:06',576),
  ('NIGHT','深夜班','14:00','23:36',576),
  ('STANDBY24','24小时手机待命','00:00','23:59',1440);
INSERT OR IGNORE INTO employee_schedules (staff_id, staff_name, team_name, week_range, schedule_text) VALUES
  ('118919','Justin Lu','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:24H_STANDBY; Sun:RDO'),
  ('118919','Justin Lu','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:24H_STANDBY; Sun:RDO'),
  ('132369','Albee Liu','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('132369','Albee Liu','Bella Team','5.4-5.10','Mon-Thu:1330-2306; Fri:RDOC; Sat:RDO; Sun:RDO'),
  ('132387','Betty Zhang','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('132387','Betty Zhang','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('132920','Heather Zou','Isaac Team','4.27-5.3','Mon:1030-2006; Tue:RDO; Wed-Thu:1030-2006; Fri:PH; Sat:RDO; Sun:1030-2006'),
  ('132920','Heather Zou','Bella Team','5.4-5.10','Mon-Thu:1030-2006; Fri:RDO; Sat:RDO; Sun:RDO'),
  ('135320','Simon Kok','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('135320','Simon Kok','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('111071','Sonia Song','Isaac Team','4.27-5.3','Mon:RDO; Tue-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:1330-2306'),
  ('111071','Sonia Song','Bella Team','5.4-5.10','Mon-Thu:1330-2306; Fri:RDO; Sat:RDO; Sun:1330-2306'),
  ('142035','Ashley Lei','Isaac Team','4.27-5.3','Mon-Tue:1330-2306; Wed:RDO; Thu:RDO; Fri-Sun:1330-2306'),
  ('142035','Ashley Lei','Bella Team','5.4-5.10','Mon-Wed:1330-2306; Thu:RDO; Fri:RDO; Sat-Sun:1330-2306'),
  ('132922','Isaac Su','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('132922','Isaac Su','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('128943','Kalei Kong','Isaac Team','4.27-5.3','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu-Sun:1100-2036'),
  ('128943','Kalei Kong','Bella Team','5.4-5.10','Mon-Fri:1100-2036; Sat:RDO; Sun:RDO'),
  ('134846','Simon Wu','Isaac Team','4.27-5.3','Mon-Tue:1400-2336; Wed:1100-2036; Thu:0930-1906; Fri:PH; Sat:RDO; Sun:RDO'),
  ('134846','Simon Wu','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed:AL; Thu:AL; Fri:RDO; Sat:RDO; Sun:1400-2336'),
  ('137291','Emily Li','Isaac Team','4.27-5.3','Mon-Tue:1100-2036; Wed:RDO; Thu:RDO; Fri-Sun:1100-2036'),
  ('137291','Emily Li','Bella Team','5.4-5.10','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu-Sun:1100-2036'),
  ('137420','Max Wang','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed:PHCL; Thu:PHCL; Fri:PH; Sat-Sun:1400-2336'),
  ('137420','Max Wang','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1400-2336'),
  ('139407','Stacey Pong','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Sun:1100-2036'),
  ('139407','Stacey Pong','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Thu:1100-2036; Fri-Sat:1400-2336; Sun:1100-2036'),
  ('140957','Owen Liang','Isaac Team','4.27-5.3','Mon:PHCL; Tue:RDO; Wed:RDO; Thu:1400-2336; Fri-Sun:1100-2036'),
  ('140957','Owen Liang','Bella Team','5.4-5.10','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu:1100-2036; Fri-Sat:1400-2336; Sun:1100-2036'),
  ('132921','Bella Guo','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed:PHCL; Thu-Sun:1330-2306'),
  ('132921','Bella Guo','Bella Team','5.4-5.10','Mon:1330-2306; Tue:RDO; Wed-Sat:1330-2306; Sun:RDO'),
  ('132386','Elva Ao','Isaac Team','4.27-5.3','Mon-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132386','Elva Ao','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed:RDOC; Thu-Sun:1330-2306'),
  ('132923','Sky Wang','Isaac Team','4.27-5.3','Mon-Thu:AL; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132923','Sky Wang','Bella Team','5.4-5.10','Mon-Fri:1030-2006; Sat:RDO; Sun:RDO'),
  ('132924','William Chen','Isaac Team','4.27-5.3','Mon-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132924','William Chen','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('139436','Joyce Yi','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed:AL; Thu-Sun:1330-2306'),
  ('139436','Joyce Yi','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('141898','Leah Zhou','Isaac Team','4.27-5.3','Mon-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:RDO'),
  ('141898','Leah Zhou','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('142133','Jeremy Cai','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('142133','Jeremy Cai','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('132945','Vicky Yue','Isaac Team','4.27-5.3','Mon-Thu:1200-2136; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132945','Vicky Yue','Bella Team','5.4-5.10','Mon-Fri:1200-2136; Sat:RDO; Sun:RDO'),
  ('140672','SiSi Sou','Isaac Team','4.27-5.3','Mon:1400-2336; Tue:RDO; Wed-Thu:1400-2336; Fri:PH; Sat-Sun:1400-2336'),
  ('140672','SiSi Sou','Bella Team','5.4-5.10','Mon-Fri:1400-2336; Sat:RDO; Sun:RDO'),
  ('141780','Sammi Xian','Isaac Team','4.27-5.3','Mon:1100-2036; Tue:RDO; Wed:RV; Thu:RV; Fri:PH; Sat-Sun:1100-2036'),
  ('141780','Sammi Xian','Bella Team','5.4-5.10','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu-Sun:1100-2036'),
  ('141906','Duke Sui','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Thu:1200-2136; Fri:PH; Sat-Sun:1200-2136'),
  ('141906','Duke Sui','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1200-2136');
INSERT OR IGNORE INTO users (id, account, password, role, display_name)
SELECT CAST(staff_id AS INTEGER), staff_id, '123456a', 'employee', staff_name
FROM employee_schedules
GROUP BY staff_id, staff_name;`
	if _, err := database.Exec(schema); err != nil {
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
