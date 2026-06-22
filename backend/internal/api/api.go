package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"precious-metal-dashboard/backend/internal/auth"
	"precious-metal-dashboard/backend/internal/config"
	"precious-metal-dashboard/backend/internal/prices"
)

type API struct {
	db     *sql.DB
	config config.Config
	prices *prices.Registry
}
type contextKey string

const userIDKey contextKey = "userID"

type userResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}
type metal struct {
	ID             int64      `json:"id"`
	Type           string     `json:"type"`
	Code           *string    `json:"code"`
	Brand          string     `json:"brand"`
	BoughtPrice    float64    `json:"bought_price"`
	BuyPrice       float64    `json:"buy_price"`
	BuybackPrice   float64    `json:"buyback_price"`
	Weight         float64    `json:"weight"`
	PriceUpdatedAt *time.Time `json:"price_updated_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func New(db *sql.DB, cfg config.Config) http.Handler {
	a := &API{db: db, config: cfg, prices: prices.NewRegistry()}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", a.register)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("POST /api/auth/refresh", a.refresh)
	mux.HandleFunc("POST /api/auth/logout", a.logout)
	mux.Handle("GET /api/me", a.requireAuth(http.HandlerFunc(a.me)))
	mux.Handle("GET /api/metals", a.requireAuth(http.HandlerFunc(a.listMetals)))
	mux.Handle("POST /api/metals", a.requireAuth(http.HandlerFunc(a.createMetal)))
	mux.Handle("GET /api/metals/{id}", a.requireAuth(http.HandlerFunc(a.getMetal)))
	mux.Handle("PUT /api/metals/{id}", a.requireAuth(http.HandlerFunc(a.updateMetal)))
	mux.Handle("DELETE /api/metals/{id}", a.requireAuth(http.HandlerFunc(a.deleteMetal)))
	mux.Handle("POST /api/metals/{id}/refresh-price", a.requireAuth(http.HandlerFunc(a.refreshPrice)))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	return a.cors(a.recoverer(mux))
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.Username, input.FullName = strings.TrimSpace(input.Username), strings.TrimSpace(input.FullName)
	if len(input.Username) < 3 || len(input.Password) < 8 || input.FullName == "" {
		fail(w, 422, "username must be at least 3 characters, password 8 characters, and full name is required")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	result, err := a.db.Exec(`INSERT INTO users(username,password_hash,full_name) VALUES(?,?,?)`, input.Username, string(hash), input.FullName)
	if err != nil {
		fail(w, 409, "username is already registered")
		return
	}
	id, _ := result.LastInsertId()
	a.issueSession(w, id, input.Username, 201)
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input struct{ Username, Password string }
	if !decode(w, r, &input) {
		return
	}
	var id int64
	var username, hash string
	err := a.db.QueryRow(`SELECT id,username,password_hash FROM users WHERE username=?`, strings.TrimSpace(input.Username)).Scan(&id, &username, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		fail(w, 401, "invalid username or password")
		return
	}
	a.issueSession(w, id, username, 200)
}

func (a *API) issueSession(w http.ResponseWriter, userID int64, username string, status int) {
	access, _ := auth.CreateAccessToken(userID, username, a.config.JWTSecret, a.config.AccessTTL)
	refresh, hash, err := auth.NewRefreshToken()
	if err != nil {
		fail(w, 500, "could not create session")
		return
	}
	_, err = a.db.Exec(`INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES(?,?,?)`, userID, hash, time.Now().Add(a.config.RefreshTTL))
	if err != nil {
		fail(w, 500, "could not save session")
		return
	}
	writeJSON(w, status, map[string]any{"access_token": access, "refresh_token": refresh, "expires_in": int(a.config.AccessTTL.Seconds())})
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decode(w, r, &input) {
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "could not refresh session")
		return
	}
	defer tx.Rollback()
	var tokenID, userID int64
	var username string
	err = tx.QueryRow(`SELECT rt.id,rt.user_id,u.username FROM refresh_tokens rt JOIN users u ON u.id=rt.user_id WHERE rt.token_hash=? AND rt.revoked_at IS NULL AND rt.expires_at>CURRENT_TIMESTAMP`, auth.HashRefreshToken(input.RefreshToken)).Scan(&tokenID, &userID, &username)
	if err != nil {
		fail(w, 401, "invalid or expired refresh token")
		return
	}
	if _, err = tx.Exec(`UPDATE refresh_tokens SET revoked_at=CURRENT_TIMESTAMP WHERE id=?`, tokenID); err != nil {
		fail(w, 500, "could not rotate session")
		return
	}
	plain, hash, err := auth.NewRefreshToken()
	if err != nil {
		fail(w, 500, "could not rotate session")
		return
	}
	if _, err = tx.Exec(`INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES(?,?,?)`, userID, hash, time.Now().Add(a.config.RefreshTTL)); err != nil {
		fail(w, 500, "could not rotate session")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "could not rotate session")
		return
	}
	access, _ := auth.CreateAccessToken(userID, username, a.config.JWTSecret, a.config.AccessTTL)
	writeJSON(w, 200, map[string]any{"access_token": access, "refresh_token": plain, "expires_in": int(a.config.AccessTTL.Seconds())})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if decode(w, r, &in) {
		_, _ = a.db.Exec(`UPDATE refresh_tokens SET revoked_at=CURRENT_TIMESTAMP WHERE token_hash=? AND revoked_at IS NULL`, auth.HashRefreshToken(in.RefreshToken))
		w.WriteHeader(204)
	}
}
func (a *API) me(w http.ResponseWriter, r *http.Request) {
	var u userResponse
	err := a.db.QueryRow(`SELECT id,username,full_name FROM users WHERE id=?`, userID(r)).Scan(&u.ID, &u.Username, &u.FullName)
	if err != nil {
		fail(w, 404, "user not found")
		return
	}
	writeJSON(w, 200, u)
}

func (a *API) listMetals(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`SELECT id,type,code,brand,bought_price,buy_price,buyback_price,weight,price_updated_at,created_at,updated_at FROM precious_metals WHERE user_id=? ORDER BY updated_at DESC`, userID(r))
	if err != nil {
		fail(w, 500, "could not load metals")
		return
	}
	defer rows.Close()
	items := []metal{}
	for rows.Next() {
		var m metal
		if scanMetal(rows.Scan, &m) == nil {
			items = append(items, m)
		}
	}
	writeJSON(w, 200, items)
}
func (a *API) getMetal(w http.ResponseWriter, r *http.Request) {
	m, err := a.findMetal(userID(r), r.PathValue("id"))
	if err != nil {
		fail(w, 404, "precious metal not found")
		return
	}
	writeJSON(w, 200, m)
}

func (a *API) createMetal(w http.ResponseWriter, r *http.Request) {
	var m metal
	if !decode(w, r, &m) || !validMetal(w, m) {
		return
	}
	quote, err := a.prices.Quote(r.Context(), m.Brand, m.Type, m.Weight)
	if err != nil {
		fail(w, 502, err.Error())
		return
	}
	code := nullableTrimmedString(m.Code)
	result, err := a.db.Exec(`INSERT INTO precious_metals(user_id,type,code,brand,bought_price,buy_price,buyback_price,weight,price_updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, userID(r), strings.TrimSpace(m.Type), code, strings.TrimSpace(m.Brand), m.BoughtPrice, quote.BuyPrice, quote.BuybackPrice, m.Weight, quote.RetrievedAt)
	if err != nil {
		fail(w, 500, "could not create precious metal")
		return
	}
	id, _ := result.LastInsertId()
	created, _ := a.findMetal(userID(r), strconv.FormatInt(id, 10))
	writeJSON(w, 201, created)
}
func (a *API) updateMetal(w http.ResponseWriter, r *http.Request) {
	var m metal
	if !decode(w, r, &m) || !validMetal(w, m) {
		return
	}
	quote, err := a.prices.Quote(r.Context(), m.Brand, m.Type, m.Weight)
	if err != nil {
		fail(w, 502, err.Error())
		return
	}
	code := nullableTrimmedString(m.Code)
	result, err := a.db.Exec(`UPDATE precious_metals SET type=?,code=?,brand=?,bought_price=?,buy_price=?,buyback_price=?,weight=?,price_updated_at=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, strings.TrimSpace(m.Type), code, strings.TrimSpace(m.Brand), m.BoughtPrice, quote.BuyPrice, quote.BuybackPrice, m.Weight, quote.RetrievedAt, r.PathValue("id"), userID(r))
	if err != nil {
		fail(w, 500, "could not update precious metal")
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		fail(w, 404, "precious metal not found")
		return
	}
	updated, _ := a.findMetal(userID(r), r.PathValue("id"))
	writeJSON(w, 200, updated)
}
func (a *API) deleteMetal(w http.ResponseWriter, r *http.Request) {
	result, _ := a.db.Exec(`DELETE FROM precious_metals WHERE id=? AND user_id=?`, r.PathValue("id"), userID(r))
	n, _ := result.RowsAffected()
	if n == 0 {
		fail(w, 404, "precious metal not found")
		return
	}
	w.WriteHeader(204)
}
func (a *API) refreshPrice(w http.ResponseWriter, r *http.Request) {
	m, err := a.findMetal(userID(r), r.PathValue("id"))
	if err != nil {
		fail(w, 404, "precious metal not found")
		return
	}
	quote, err := a.prices.Quote(r.Context(), m.Brand, m.Type, m.Weight)
	if err != nil {
		fail(w, 502, err.Error())
		return
	}
	_, err = a.db.Exec(`UPDATE precious_metals SET buy_price=?,buyback_price=?,price_updated_at=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, quote.BuyPrice, quote.BuybackPrice, quote.RetrievedAt, m.ID, userID(r))
	if err != nil {
		fail(w, 500, "could not save price")
		return
	}
	updated, _ := a.findMetal(userID(r), r.PathValue("id"))
	writeJSON(w, 200, updated)
}

func (a *API) findMetal(uid int64, id string) (metal, error) {
	var m metal
	err := scanMetal(a.db.QueryRow(`SELECT id,type,code,brand,bought_price,buy_price,buyback_price,weight,price_updated_at,created_at,updated_at FROM precious_metals WHERE id=? AND user_id=?`, id, uid).Scan, &m)
	return m, err
}
func scanMetal(scan func(...any) error, m *metal) error {
	return scan(&m.ID, &m.Type, &m.Code, &m.Brand, &m.BoughtPrice, &m.BuyPrice, &m.BuybackPrice, &m.Weight, &m.PriceUpdatedAt, &m.CreatedAt, &m.UpdatedAt)
}
func validMetal(w http.ResponseWriter, m metal) bool {
	if strings.TrimSpace(m.Type) == "" || strings.TrimSpace(m.Brand) == "" || m.BoughtPrice < 0 || m.BuyPrice < 0 || m.BuybackPrice < 0 || m.Weight <= 0 {
		fail(w, 422, "type, brand, and positive weight are required; prices cannot be negative")
		return false
	}
	return true
}
func nullableTrimmedString(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func (a *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Bearer(r.Header.Get("Authorization"))
		if err != nil {
			fail(w, 401, "authentication required")
			return
		}
		claims, err := auth.ParseAccessToken(token, a.config.JWTSecret)
		if err != nil {
			fail(w, 401, "invalid or expired access token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, claims.Sub)))
	})
}
func userID(r *http.Request) int64 { return r.Context().Value(userIDKey).(int64) }
func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", a.config.AllowedOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				log.Printf("panic: %v", value)
				fail(w, 500, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		fail(w, 400, "invalid JSON body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
