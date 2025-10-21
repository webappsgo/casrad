package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/casapps/casrad/internal/auth"
	"github.com/casapps/casrad/internal/database"
	"github.com/casapps/casrad/internal/protocols"
	"github.com/gorilla/mux"
)

type HTTPServer struct {
	Port       int
	db         *database.Engine
	authMgr    *auth.AuthManager
	router     *mux.Router
	server     *http.Server
	webAssets  embed.FS
	templates  embed.FS
	setupMode  bool
}

func New(port int, db *database.Engine, webAssets, templates embed.FS) *HTTPServer {
	s := &HTTPServer{
		Port:      port,
		db:        db,
		authMgr:   auth.NewAuthManager(db),
		webAssets: webAssets,
		templates: templates,
	}

	// Auto-select port if needed
	if s.Port == 0 {
		s.Port = s.findAvailablePort()
	}

	s.setupRoutes()
	return s
}

func (s *HTTPServer) findAvailablePort() int {
	// Try ports in range 64000-64999
	for port := 64000; port <= 64999; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	// Fallback to any available port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return 8080
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func (s *HTTPServer) EnableSetupMode() {
	s.setupMode = true
}

func (s *HTTPServer) setupRoutes() {
	s.router = mux.NewRouter()

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.apiMiddleware)

	// Auth endpoints
	api.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
	api.HandleFunc("/auth/logout", s.handleLogout).Methods("POST")
	api.HandleFunc("/auth/register", s.handleRegister).Methods("POST")
	api.HandleFunc("/auth/session", s.handleSessionCheck).Methods("GET")

	// Setup wizard (only in setup mode)
	if s.setupMode {
		api.HandleFunc("/setup/check", s.handleSetupCheck).Methods("GET")
		api.HandleFunc("/setup/complete", s.handleSetupComplete).Methods("POST")
	}

	// Admin endpoints
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(s.requireAuth, s.requireAdmin)
	admin.HandleFunc("/dashboard", s.handleAdminDashboard).Methods("GET")
	admin.HandleFunc("/users", s.handleAdminUsers).Methods("GET")
	admin.HandleFunc("/settings", s.handleAdminSettings).Methods("GET", "POST")

	// Library endpoints
	api.HandleFunc("/tracks", s.handleGetTracks).Methods("GET")
	api.HandleFunc("/albums", s.handleGetAlbums).Methods("GET")
	api.HandleFunc("/albums/{id}/tracks", s.handleGetAlbumTracks).Methods("GET")
	api.HandleFunc("/artists", s.handleGetArtists).Methods("GET")
	api.HandleFunc("/playlists", s.handleGetPlaylists).Methods("GET")
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
	api.HandleFunc("/library/scan", s.handleScanLibrary).Methods("POST")

	// Streaming and playback
	api.HandleFunc("/stream/{id}", s.handleStream).Methods("GET")
	api.HandleFunc("/tracks/{id}/play", s.handleTrackPlay).Methods("POST")

	// Protocol APIs will be registered separately
	s.router.PathPrefix("/ampache/server").Handler(http.HandlerFunc(s.handleAmpache))
	s.router.PathPrefix("/webdav").Handler(http.HandlerFunc(s.handleWebDAV))

	// Static files
	staticFS, err := fs.Sub(s.webAssets, "static")
	if err != nil {
		log.Printf("Warning: static files not embedded properly: %v", err)
		// Fallback - serve directly
		s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
			http.FileServer(http.FS(s.webAssets))))
	} else {
		s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
			http.FileServer(http.FS(staticFS))))
	}

	// Main web UI
	s.router.PathPrefix("/").HandlerFunc(s.handleWebUI)
}

func (s *HTTPServer) Start() error {
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("HTTP server listening on port %d", s.Port)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Middleware
func (s *HTTPServer) apiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// CORS headers for API
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *HTTPServer) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check cookie for session
		cookie, err := r.Cookie("session")
		if err == nil {
			userID, err := s.authMgr.ValidateSession(cookie.Value)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), "userID", userID))
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check authorization header for API token
		if token := r.Header.Get("Authorization"); token != "" {
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			userID, _, err := s.authMgr.ValidateAPIToken(token)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), "userID", userID))
				next.ServeHTTP(w, r)
				return
			}
		}

		// Unauthorized
		s.sendError(w, http.StatusUnauthorized, "Authentication required")
	})
}

func (s *HTTPServer) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID")
		if userID == nil {
			s.sendError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		// Get user and check role
		user, err := s.authMgr.GetUser(userID.(int))
		if err != nil || !user.IsAdmin() {
			s.sendError(w, http.StatusForbidden, "Admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler implementations
func (s *HTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Authenticate user
	userID, err := s.authMgr.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		s.sendJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Create session
	sessionID, err := s.authMgr.CreateSession(userID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	// Get user info
	user, _ := s.authMgr.GetUser(userID)

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"user":     user,
		"sessionID": sessionID,
	})
}

func (s *HTTPServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie
	cookie, err := r.Cookie("session")
	if err == nil {
		s.authMgr.DestroySession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out",
	})
}

func (s *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Check password strength
	if err := s.authMgr.PasswordStrength(req.Password); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Determine role (first user is admin)
	var userCount int
	s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	role := "user"
	if userCount == 0 {
		role = "admin"
	}

	// Create user
	if err := s.authMgr.CreateUser(req.Username, req.Email, req.Password, role); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Registration successful",
	})
}

func (s *HTTPServer) handleSessionCheck(w http.ResponseWriter, r *http.Request) {
	// Check cookie for session
	cookie, err := r.Cookie("session")
	if err != nil {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Validate session
	userID, err := s.authMgr.ValidateSession(cookie.Value)
	if err != nil {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Get user info
	user, _ := s.authMgr.GetUser(userID)

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"user":          user,
	})
}

func (s *HTTPServer) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	var setupCompleted bool
	err := s.db.QueryRow("SELECT setup_completed FROM setup_state WHERE id = 1").Scan(&setupCompleted)

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"setupRequired": err != nil || !setupCompleted,
		"step":          0,
	})
}

func (s *HTTPServer) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement setup completion
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Setup completed",
	})
}

func (s *HTTPServer) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement dashboard data gathering
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"users":       0,
		"tracks":      0,
		"streams":     0,
		"diskUsage":   0,
		"uptime":      0,
	})
}

func (s *HTTPServer) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user management
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"users": []interface{}{},
	})
}

func (s *HTTPServer) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// TODO: Get settings
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"settings": map[string]interface{}{},
		})
	} else {
		// TODO: Update settings
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}


func (s *HTTPServer) RegisterSubsonicAPI(subsonic *protocols.SubsonicServer) {
	s.router.PathPrefix("/subsonic/rest").Handler(subsonic)
}

func (s *HTTPServer) RegisterAmpacheAPI(ampache *protocols.AmpacheServer) {
	s.router.PathPrefix("/ampache/server").Handler(ampache)
}

func (s *HTTPServer) RegisterWebDAV(webdav *protocols.WebDAVServer) {
	s.router.PathPrefix("/webdav").Handler(webdav)
}

func (s *HTTPServer) RegisterDLNA(dlna *protocols.DLNAServer) {
	s.router.PathPrefix("/dlna").Handler(dlna)
}

func (s *HTTPServer) handleAmpache(w http.ResponseWriter, r *http.Request) {
	// Handled by RegisterAmpacheAPI
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintln(w, "Ampache API not registered")
}

func (s *HTTPServer) handleWebDAV(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement WebDAV
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintln(w, "WebDAV not yet implemented")
}

func (s *HTTPServer) handleWebUI(w http.ResponseWriter, r *http.Request) {
	// Check if setup is required
	if s.setupMode {
		// Check if this is the setup page request
		if r.URL.Path == "/setup" || r.URL.Path == "/" {
			data, err := s.templates.ReadFile("web/templates/setup.html")
			if err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write(data)
				return
			}
		}
	}

	// Check if this is admin page
	if r.URL.Path == "/admin" {
		data, err := s.templates.ReadFile("web/templates/admin.html")
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
	}

	// Serve the main interface
	data, err := s.templates.ReadFile("web/templates/index.html")
	if err != nil {
		// Fallback to simple HTML if template not found
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CASRAD - Complete Audio Streaming</title>
    <link rel="stylesheet" href="/static/css/dracula.css">
</head>
<body>
    <div class="container">
        <h1>CASRAD</h1>
        <p>Loading...</p>
    </div>
    <audio id="audio-player" preload="metadata"></audio>
    <script src="/static/js/casrad.js"></script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Helper functions
func (s *HTTPServer) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) sendError(w http.ResponseWriter, status int, message string) {
	s.sendJSON(w, status, map[string]interface{}{
		"error": message,
	})
}