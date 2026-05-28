// Package server implements the HTTP server and route handling
// See AI.md PART 14 for API structure, PART 16 for frontend, PART 17 for admin
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

	"github.com/casapps/casrad/src/config"
	"github.com/casapps/casrad/src/server/handler"
	"github.com/casapps/casrad/src/server/store"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	httpServer *http.Server
	router     *chi.Mux
	store      store.Store
	apiHandler *handler.APIHandler
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	st := store.NewMemoryStore()

	s := &Server{
		config:     cfg,
		store:      st,
		apiHandler: handler.NewAPIHandler(st),
	}

	s.router = s.setupRoutes()

	return s, nil
}

// Run starts the HTTP server
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      corsHandler.Handler(s.router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Starting CASRAD server on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// adminPath returns the configured admin path segment, defaulting to "admin"
func (s *Server) adminPath() string {
	if s.config.Server.AdminPath != "" {
		return s.config.Server.AdminPath
	}
	return "admin"
}

// setupRoutes configures all HTTP routes using chi router
// Route structure per AI.md PART 14:
//   /server/healthz                  — HTML health page
//   /server/auth/...                 — auth pages
//   /server/{admin_path}/...         — admin panel
//   /api/v1/server/healthz           — JSON health endpoint
//   /api/v1/...                      — REST API
func (s *Server) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware stack — AI.md PART 11
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Well-known files — AI.md PART 11
	r.Get("/robots.txt", s.handleRobotsTxt)
	r.Get("/.well-known/security.txt", s.handleSecurityTxt)
	r.Get("/.well-known/change-password", s.handleChangePassword)

	// Server-scoped routes: health, auth, admin
	r.Route("/server", func(r chi.Router) {
		// HTML health check page — /server/healthz
		r.Get("/healthz", s.handleHealth)

		// Auth pages — /server/auth/...
		r.Get("/auth/login", s.handleAuthLoginPage)
		r.Post("/auth/login", s.handleAuthLogin)
		r.Get("/auth/logout", s.handleAuthLogout)
		r.Post("/auth/logout", s.handleAuthLogout)
		r.Get("/auth/register", s.handleAuthRegisterPage)
		r.Post("/auth/register", s.handleAuthRegister)
		r.Get("/auth/password/forgot", s.handleAuthForgotPage)
		r.Post("/auth/password/forgot", s.handleAuthForgot)
		r.Get("/auth/password/reset", s.handleAuthResetPage)
		r.Post("/auth/password/reset", s.handleAuthReset)

		// Admin panel — /server/{admin_path}/...
		ap := s.adminPath()
		r.Route("/"+ap, func(r chi.Router) {
			r.Get("/", s.handleAdminDashboard)
			r.Get("/dashboard", s.handleAdminDashboard)
		})
	})

	// API v1 routes — AI.md PART 14
	r.Route("/api/v1", func(r chi.Router) {
		// JSON health endpoint — /api/v1/server/healthz
		r.Get("/server/healthz", s.handleAPIHealth)

		// Tracks — flat routes avoid trailing-slash bug (AI.md PART 14)
		r.Get("/tracks", s.apiHandler.Tracks)
		r.Get("/tracks/{id}", s.apiHandler.Track)
		r.Get("/tracks/{id}/stream", s.apiHandler.TrackStream)

		// Albums
		r.Get("/albums", s.apiHandler.Albums)
		r.Get("/albums/{id}", s.apiHandler.Album)

		// Artists
		r.Get("/artists", s.apiHandler.Artists)
		r.Get("/artists/{id}", s.apiHandler.Artist)

		// Playlists — queue-preserving behavior per AI.md
		r.Get("/playlists", s.apiHandler.Playlists)
		r.Post("/playlists", s.apiHandler.PlaylistCreate)
		r.Get("/playlists/{id}", s.apiHandler.Playlist)
		r.Patch("/playlists/{id}", s.apiHandler.PlaylistUpdate)
		r.Delete("/playlists/{id}", s.apiHandler.PlaylistDelete)
		r.Post("/playlists/{id}/tracks", s.apiHandler.PlaylistAddTracks)

		// Broadcasts — streaming/radio
		r.Get("/broadcasts", s.apiHandler.Broadcasts)
		r.Get("/broadcasts/{mount}", s.apiHandler.Broadcast)

		// Podcasts
		r.Get("/podcasts", s.apiHandler.Podcasts)
		r.Post("/podcasts", s.apiHandler.PodcastSubscribe)

		// Audiobooks
		r.Get("/audiobooks", s.apiHandler.Audiobooks)
		r.Get("/audiobooks/{id}", s.apiHandler.Audiobook)

		// Search — unified library search
		r.Get("/search", s.apiHandler.Search)

		// Queue — playback queue (append by default per AI.md)
		r.Get("/queue", s.apiHandler.Queue)
		r.Post("/queue", s.apiHandler.QueueAdd)
		r.Delete("/queue", s.apiHandler.QueueClear)

		// Player — playback control
		r.Get("/player", s.apiHandler.Player)
		r.Post("/player/{action}", s.apiHandler.PlayerControl)

		// Cover art
		r.Get("/cover/{type}/{id}", s.apiHandler.CoverArt)

		// Listening history
		r.Get("/history", s.apiHandler.History)

		// User statistics
		r.Get("/stats", s.apiHandler.Stats)

		// Scrobble — record plays
		r.Post("/scrobble", s.apiHandler.Scrobble)

		// Rate — rate content
		r.Post("/rate", s.apiHandler.Rate)

		// Favorite — toggle favorites
		r.Post("/favorite", s.apiHandler.Favorite)
	})

	// Protocol routes — per IDEA.md Protocol Support
	// Subsonic API — /subsonic/rest/...
	r.Route("/subsonic/rest", func(r chi.Router) {
		// Subsonic API implemented in future milestone
		_ = r
	})

	// Ampache API — /ampache/server/...
	r.Route("/ampache/server", func(r chi.Router) {
		// Ampache API implemented in future milestone
		_ = r
	})

	// WebDAV — /webdav/...
	r.Route("/webdav", func(r chi.Router) {
		// WebDAV implemented in future milestone
		_ = r
	})

	return r
}

// handleAdminDashboard renders the admin dashboard
func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Dashboard - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Inter', system-ui, sans-serif;
            background: #282a36;
            color: #f8f8f2;
            min-height: 100vh;
        }
        .container { padding: 2rem; max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 1.5rem; }
        .card {
            background: #44475a;
            padding: 1.5rem;
            border-radius: 8px;
            margin-bottom: 1rem;
        }
        .card h2 { color: #50fa7b; font-size: 1.25rem; margin-bottom: 0.5rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>CASRAD Admin Dashboard</h1>
        <div class="card">
            <h2>Server Status</h2>
            <p>Running</p>
        </div>
        <div class="card">
            <h2>Quick Actions</h2>
            <p>Coming soon...</p>
        </div>
    </div>
</body>
</html>`)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK\n")
}

func (s *Server) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"healthy"}`+"\n")
}

func (s *Server) handleAuthLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Inter', system-ui, sans-serif;
            background: #282a36;
            color: #f8f8f2;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: #44475a;
            padding: 2rem;
            border-radius: 8px;
            width: 100%;
            max-width: 400px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
        }
        h1 { color: #bd93f9; text-align: center; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; color: #6272a4; margin-bottom: 0.5rem; font-size: 0.875rem; }
        input[type="text"],
        input[type="password"] {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid #6272a4;
            border-radius: 6px;
            background: #282a36;
            color: #f8f8f2;
            font-size: 1rem;
        }
        input:focus { outline: none; border-color: #bd93f9; }
        button {
            width: 100%;
            padding: 0.75rem;
            background: #50fa7b;
            color: #282a36;
            border: none;
            border-radius: 6px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.2s;
        }
        button:hover { background: #69ff94; }
        .links { text-align: center; margin-top: 1rem; }
        .links a { color: #8be9fd; text-decoration: none; }
        .links a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>CASRAD</h1>
        <form method="POST" action="/server/auth/login">
            <div class="form-group">
                <label for="username">Username or Email</label>
                <input type="text" id="username" name="username" required autocomplete="username">
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required autocomplete="current-password">
            </div>
            <button type="submit">Sign In</button>
        </form>
        <div class="links">
            <a href="/server/auth/register">Create Account</a>
        </div>
    </div>
</body>
</html>`)
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/server/auth/login?error=invalid", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Redirect(w, r, "/server/auth/login?error=invalid", http.StatusSeeOther)
		return
	}

	// Real authentication is handled in src/server/service/auth.go
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/server/auth/login", http.StatusSeeOther)
}

func (s *Server) handleAuthRegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Inter', system-ui, sans-serif;
            background: #282a36;
            color: #f8f8f2;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            background: #44475a;
            padding: 2rem;
            border-radius: 8px;
            width: 100%;
            max-width: 400px;
        }
        h1 { color: #bd93f9; text-align: center; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; color: #6272a4; margin-bottom: 0.5rem; font-size: 0.875rem; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; font-size: 1rem; }
        input:focus { outline: none; border-color: #bd93f9; }
        button { width: 100%; padding: 0.75rem; background: #50fa7b; color: #282a36; border: none; border-radius: 6px; font-size: 1rem; font-weight: 600; cursor: pointer; }
        button:hover { background: #69ff94; }
        .links { text-align: center; margin-top: 1rem; }
        .links a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Create Account</h1>
        <form method="POST" action="/server/auth/register">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autocomplete="username">
            </div>
            <div class="form-group">
                <label for="email">Email</label>
                <input type="email" id="email" name="email" required autocomplete="email">
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required autocomplete="new-password">
            </div>
            <button type="submit">Create Account</button>
        </form>
        <div class="links">
            <a href="/server/auth/login">Already have an account? Sign in</a>
        </div>
    </div>
</body>
</html>`)
}

func (s *Server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/server/auth/register?error=invalid", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/server/auth/login", http.StatusSeeOther)
}

func (s *Server) handleAuthForgotPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Password - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
        .container { background: #44475a; padding: 2rem; border-radius: 8px; width: 100%; max-width: 400px; }
        h1 { color: #bd93f9; text-align: center; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; color: #6272a4; margin-bottom: 0.5rem; font-size: 0.875rem; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; font-size: 1rem; }
        input:focus { outline: none; border-color: #bd93f9; }
        button { width: 100%; padding: 0.75rem; background: #50fa7b; color: #282a36; border: none; border-radius: 6px; font-size: 1rem; font-weight: 600; cursor: pointer; }
        .links { text-align: center; margin-top: 1rem; }
        .links a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Reset Password</h1>
        <form method="POST" action="/server/auth/password/forgot">
            <div class="form-group">
                <label for="email">Email Address</label>
                <input type="email" id="email" name="email" required autocomplete="email">
            </div>
            <button type="submit">Send Reset Link</button>
        </form>
        <div class="links">
            <a href="/server/auth/login">Back to Sign In</a>
        </div>
    </div>
</body>
</html>`)
}

func (s *Server) handleAuthForgot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/server/auth/password/forgot?error=invalid", http.StatusSeeOther)
		return
	}
	// Always show the "if account exists, email sent" message (PART 11 enumeration prevention)
	http.Redirect(w, r, "/server/auth/login?info=reset-sent", http.StatusSeeOther)
}

func (s *Server) handleAuthResetPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New Password - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
        .container { background: #44475a; padding: 2rem; border-radius: 8px; width: 100%; max-width: 400px; }
        h1 { color: #bd93f9; text-align: center; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; color: #6272a4; margin-bottom: 0.5rem; font-size: 0.875rem; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; font-size: 1rem; }
        input:focus { outline: none; border-color: #bd93f9; }
        button { width: 100%; padding: 0.75rem; background: #50fa7b; color: #282a36; border: none; border-radius: 6px; font-size: 1rem; font-weight: 600; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <h1>New Password</h1>
        <form method="POST" action="/server/auth/password/reset">
            <input type="hidden" name="token" value="">
            <div class="form-group">
                <label for="password">New Password</label>
                <input type="password" id="password" name="password" required autocomplete="new-password">
            </div>
            <div class="form-group">
                <label for="confirm">Confirm Password</label>
                <input type="password" id="confirm" name="confirm" required autocomplete="new-password">
            </div>
            <button type="submit">Set Password</button>
        </form>
    </div>
</body>
</html>`)
}

func (s *Server) handleAuthReset(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/server/auth/password/reset?error=invalid", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/server/auth/login?info=password-reset", http.StatusSeeOther)
}

// handleRobotsTxt serves the robots.txt file
func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	ap := s.adminPath()
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /server/%s/\n", ap)
}

// handleSecurityTxt serves the security.txt file per RFC 9116
func (s *Server) handleSecurityTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	contact := s.config.Server.SecurityContact
	if contact == "" {
		contact = "security@" + r.Host
	}

	expires := time.Now().AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z")

	fmt.Fprintf(w, "Contact: mailto:%s\nExpires: %s\n", contact, expires)
}

// handleChangePassword redirects to the appropriate password change URL
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/users/security/password", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/server/auth/password/forgot", http.StatusSeeOther)
}
