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
	// Create store (using memory store for now - per AI.md PART 3)
	st := store.NewMemoryStore()

	s := &Server{
		config:     cfg,
		store:      st,
		apiHandler: handler.NewAPIHandler(st),
	}

	// Setup routes
	s.router = s.setupRoutes()

	return s, nil
}

// Run starts the HTTP server
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)

	// Wrap with CORS handler - AI.md PART 3
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

// setupRoutes configures all HTTP routes using chi router
// See AI.md PART 14 for route structure
func (s *Server) setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware stack - AI.md PART 11
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Well-known files - See AI.md PART 11
	r.Get("/robots.txt", s.handleRobotsTxt)
	r.Get("/.well-known/security.txt", s.handleSecurityTxt)
	r.Get("/.well-known/change-password", s.handleChangePassword)

	// Health check - See AI.md system routes
	r.Get("/healthz", s.handleHealth)
	r.Get("/version", s.handleVersion)

	// Auth routes - See AI.md PART 14
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", s.handleAuthLoginPage)
		r.Post("/login", s.handleAuthLogin)
		r.Get("/logout", s.handleAuthLogout)
		r.Post("/logout", s.handleAuthLogout)
	})

	// API v1 routes - See AI.md PART 14
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/healthz", s.handleAPIHealth)

		// Tracks - Music library
		r.Route("/tracks", func(r chi.Router) {
			r.Get("/", s.apiHandler.Tracks)
			r.Get("/{id}", s.apiHandler.Track)
			r.Get("/{id}/stream", s.apiHandler.TrackStream)
		})

		// Albums
		r.Route("/albums", func(r chi.Router) {
			r.Get("/", s.apiHandler.Albums)
			r.Get("/{id}", s.apiHandler.Album)
		})

		// Artists
		r.Route("/artists", func(r chi.Router) {
			r.Get("/", s.apiHandler.Artists)
			r.Get("/{id}", s.apiHandler.Artist)
		})

		// Playlists - per AI.md queue-preserving behavior
		r.Route("/playlists", func(r chi.Router) {
			r.Get("/", s.apiHandler.Playlists)
			r.Post("/", s.apiHandler.PlaylistCreate)
			r.Get("/{id}", s.apiHandler.Playlist)
			r.Patch("/{id}", s.apiHandler.PlaylistUpdate)
			r.Delete("/{id}", s.apiHandler.PlaylistDelete)
			r.Post("/{id}/tracks", s.apiHandler.PlaylistAddTracks)
		})

		// Broadcasts - Streaming/Radio
		r.Route("/broadcasts", func(r chi.Router) {
			r.Get("/", s.apiHandler.Broadcasts)
			r.Get("/{mount}", s.apiHandler.Broadcast)
		})

		// Podcasts
		r.Route("/podcasts", func(r chi.Router) {
			r.Get("/", s.apiHandler.Podcasts)
			r.Post("/", s.apiHandler.PodcastSubscribe)
		})

		// Audiobooks
		r.Route("/audiobooks", func(r chi.Router) {
			r.Get("/", s.apiHandler.Audiobooks)
			r.Get("/{id}", s.apiHandler.Audiobook)
		})

		// Search - unified search across library
		r.Get("/search", s.apiHandler.Search)

		// Queue - playback queue management (append by default per AI.md)
		r.Route("/queue", func(r chi.Router) {
			r.Get("/", s.apiHandler.Queue)
			r.Post("/", s.apiHandler.QueueAdd)
			r.Delete("/", s.apiHandler.QueueClear)
		})

		// Player - playback control
		r.Route("/player", func(r chi.Router) {
			r.Get("/", s.apiHandler.Player)
			r.Post("/{action}", s.apiHandler.PlayerControl)
		})

		// Cover art
		r.Get("/cover/{type}/{id}", s.apiHandler.CoverArt)

		// History - listening history
		r.Get("/history", s.apiHandler.History)

		// Stats - user statistics
		r.Get("/stats", s.apiHandler.Stats)

		// Scrobble - record plays
		r.Post("/scrobble", s.apiHandler.Scrobble)

		// Rate - rate content
		r.Post("/rate", s.apiHandler.Rate)

		// Favorite - toggle favorites
		r.Post("/favorite", s.apiHandler.Favorite)
	})

	// Admin routes - configured with dynamic admin path
	adminPath := s.config.Server.AdminPath
	if adminPath == "" {
		adminPath = "admin"
	}
	r.Route("/"+adminPath, func(r chi.Router) {
		// TODO: Add admin routes per AI.md PART 17
		r.Get("/", s.handleAdminDashboard)
		r.Get("/dashboard", s.handleAdminDashboard)
	})

	// Protocol routes - per IDEA.md Protocol Support
	// MPD protocol - port 6600 (separate TCP server)
	// Subsonic API - /subsonic/rest/*
	r.Route("/subsonic/rest", func(r chi.Router) {
		// TODO: Implement Subsonic API
	})

	// Ampache API - /ampache/server/*
	r.Route("/ampache/server", func(r chi.Router) {
		// TODO: Implement Ampache API
	})

	// WebDAV - /webdav/*
	r.Route("/webdav", func(r chi.Router) {
		// TODO: Implement WebDAV
	})

	return r
}

// handleAdminDashboard renders the admin dashboard
func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
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
</html>`))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK\n")
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"version":"dev"}`+"\n")
}

func (s *Server) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"healthy"}`+"\n")
}

func (s *Server) handleAuthLoginPage(w http.ResponseWriter, r *http.Request) {
	// Render login page with Dracula theme
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
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
        h1 {
            color: #bd93f9;
            text-align: center;
            margin-bottom: 1.5rem;
        }
        .form-group {
            margin-bottom: 1rem;
        }
        label {
            display: block;
            color: #6272a4;
            margin-bottom: 0.5rem;
            font-size: 0.875rem;
        }
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
        input:focus {
            outline: none;
            border-color: #bd93f9;
        }
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
        button:hover {
            background: #69ff94;
        }
        .error {
            background: rgba(255, 85, 85, 0.2);
            color: #ff5555;
            padding: 0.75rem;
            border-radius: 6px;
            margin-bottom: 1rem;
            text-align: center;
        }
        .links {
            text-align: center;
            margin-top: 1rem;
        }
        .links a {
            color: #8be9fd;
            text-decoration: none;
        }
        .links a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>CASRAD</h1>
        <form method="POST" action="/auth/login">
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
            <a href="/auth/register">Create Account</a>
        </div>
    </div>
</body>
</html>`))
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/auth/login?error=invalid", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate credentials (this is a stub - real implementation in handler/auth.go)
	if username == "" || password == "" {
		http.Redirect(w, r, "/auth/login?error=invalid", http.StatusSeeOther)
		return
	}

	// Authentication is handled by AuthHandler.Login in handler/auth.go
	// This stub redirects to dashboard on success
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// Well-known file handlers - See AI.md PART 11

// handleRobotsTxt serves the robots.txt file
func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	adminPath := s.config.Server.AdminPath
	if adminPath == "" {
		adminPath = "admin"
	}
	fmt.Fprintf(w, `# CASRAD - robots.txt
# See AI.md PART 11

User-agent: *
Allow: /
Allow: /api
Disallow: /%s
`, adminPath)
}

// handleSecurityTxt serves the security.txt file per RFC 9116
func (s *Server) handleSecurityTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Get security contact from config or use default
	contact := s.config.Server.SecurityContact
	if contact == "" {
		contact = "security@" + r.Host
	}

	// Expires 1 year from now per AI.md PART 11
	expires := time.Now().AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z")

	fmt.Fprintf(w, `# CASRAD Security Contact
# See AI.md PART 11 - RFC 9116

Contact: mailto:%s
Expires: %s
`, contact, expires)
}

// handleChangePassword redirects to appropriate password change URL
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in via session cookie
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		// Logged in - redirect to user security page
		http.Redirect(w, r, "/users/security/password", http.StatusSeeOther)
		return
	}
	// Not logged in - redirect to password forgot page
	http.Redirect(w, r, "/auth/password/forgot", http.StatusSeeOther)
}
