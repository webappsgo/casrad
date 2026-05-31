// Package handler - Authentication handlers
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/service"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	authService     *service.AuthService
	userService     *service.UserService
	emailService    *service.EmailService
	securityMW      *middleware.SecurityMiddleware
	// "disabled", "public", "private", "approval"
	registrationMode string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	auth *service.AuthService,
	user *service.UserService,
	email *service.EmailService,
	security *middleware.SecurityMiddleware,
	registrationMode string,
) *AuthHandler {
	if registrationMode == "" {
		registrationMode = "disabled"
	}
	return &AuthHandler{
		authService:      auth,
		userService:      user,
		emailService:     email,
		securityMW:       security,
		registrationMode: registrationMode,
	}
}

// LoginPage handles GET /auth/login
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	csrfToken := h.securityMW.GenerateCSRFToken()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; display: flex; justify-content: center; align-items: center; }
        .container { width: 100%; max-width: 400px; padding: 2rem; }
        h1 { text-align: center; margin-bottom: 2rem; color: #bd93f9; }
        form { display: flex; flex-direction: column; gap: 1rem; }
        label { font-size: 0.875rem; color: #6272a4; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #44475a; color: #f8f8f2; font-size: 1rem; }
        input:focus { outline: none; border-color: #bd93f9; }
        button { padding: 0.75rem; border: none; border-radius: 6px; background: #bd93f9; color: #282a36; font-size: 1rem; cursor: pointer; font-weight: 600; }
        button:hover { background: #ff79c6; }
        .error { color: #ff5555; text-align: center; margin-bottom: 1rem; }
        .link { text-align: center; margin-top: 1rem; }
        .link a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>CASRAD</h1>
        <form method="POST" action="/auth/login">
            <input type="hidden" name="csrf_token" value="` + csrfToken + `">
            <input type="hidden" name="redirect" value="` + redirect + `">
            <div>
                <label for="identifier">Username or Email</label>
                <input type="text" id="identifier" name="identifier" required autofocus>
            </div>
            <div>
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit">Sign In</button>
        </form>
    </div>
</body>
</html>`))
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	identifier := r.FormValue("identifier")
	password := r.FormValue("password")
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}

	// Get client IP
	ip := getClientIP(r)

	// Authenticate
	userID, adminID, err := h.authService.Authenticate(ctx, identifier, password, ip)
	if err != nil {
		// Return error based on accept header
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			switch err {
			case service.ErrAccountLocked:
				w.WriteHeader(http.StatusLocked)
				json.NewEncoder(w).Encode(map[string]string{"error": "account locked"})
			case service.ErrAccountDisabled:
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "account disabled"})
			default:
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
			}
			return
		}
		// Redirect back to login with error
		http.Redirect(w, r, "/auth/login?error=invalid", http.StatusSeeOther)
		return
	}

	// Create session
	sessionID, err := h.authService.CreateSession(ctx, userID, adminID, ip, r.UserAgent())
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Return JSON response for API requests
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": sessionID,
			"user_id":    userID,
			"admin_id":   adminID,
			"is_admin":   adminID != 0,
		})
		return
	}

	// Redirect to target page
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// Logout handles GET/POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session from cookie
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		h.authService.InvalidateSession(ctx, cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"logged out"}` + "\n"))
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RegisterPage handles GET /auth/register
func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if h.registrationMode == "disabled" {
		http.Error(w, "Registration is disabled", http.StatusNotFound)
		return
	}

	csrfToken := h.securityMW.GenerateCSRFToken()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; display: flex; justify-content: center; align-items: center; }
        .container { width: 100%; max-width: 400px; padding: 2rem; }
        h1 { text-align: center; margin-bottom: 2rem; color: #bd93f9; }
        form { display: flex; flex-direction: column; gap: 1rem; }
        label { font-size: 0.875rem; color: #6272a4; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #44475a; color: #f8f8f2; font-size: 1rem; }
        input:focus { outline: none; border-color: #bd93f9; }
        button { padding: 0.75rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; font-size: 1rem; cursor: pointer; font-weight: 600; }
        button:hover { background: #8be9fd; }
        .link { text-align: center; margin-top: 1rem; }
        .link a { color: #8be9fd; text-decoration: none; }
        small { color: #6272a4; font-size: 0.75rem; }
        .error { color: #ff5555; font-size: 0.75rem; margin-top: 0.25rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Create Account</h1>
        <form method="POST" action="/auth/register">
            <input type="hidden" name="csrf_token" value="` + csrfToken + `">
            <div>
                <label for="username">Username</label>
                <input type="text" id="username" name="username" pattern="^[a-zA-Z0-9_-]{3,32}$" required autocomplete="username">
                <small>3-32 characters: letters, numbers, underscores, hyphens</small>
            </div>
            <div>
                <label for="email">Email</label>
                <input type="email" id="email" name="email" required autocomplete="email">
            </div>
            <div>
                <label for="password">Password</label>
                <input type="password" id="password" name="password" minlength="8" maxlength="128" required autocomplete="new-password">
                <small>8-128 characters, cannot start/end with whitespace</small>
            </div>
            <div>
                <label for="confirm_password">Confirm Password</label>
                <input type="password" id="confirm_password" name="confirm_password" minlength="8" maxlength="128" required autocomplete="new-password">
            </div>
            <button type="submit">Create Account</button>
        </form>
        <div class="link">
            <a href="/auth/login">Already have an account? Sign in</a>
        </div>
    </div>
</body>
</html>`))
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.registrationMode == "disabled" {
		http.Error(w, "Registration is disabled", http.StatusNotFound)
		return
	}

	ctx := r.Context()

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate all registration input
	validationResult := service.ValidateRegistration(username, email, password, confirmPassword)
	if validationResult.HasErrors() {
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "validation_failed",
				"errors": validationResult.Errors,
			})
			return
		}
		// Get first error for redirect
		var firstError string
		for _, errMsg := range validationResult.Errors {
			firstError = errMsg
			break
		}
		http.Redirect(w, r, "/auth/register?error="+firstError, http.StatusSeeOther)
		return
	}

	// Use auth service for registration (includes duplicate checking)
	userID, result, err := h.authService.RegisterUser(ctx, username, email, password, confirmPassword)
	if err != nil {
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		http.Redirect(w, r, "/auth/register?error=server_error", http.StatusSeeOther)
		return
	}
	if result != nil && result.HasErrors() {
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "validation_failed",
				"errors": result.Errors,
			})
			return
		}
		var firstError string
		for _, errMsg := range result.Errors {
			firstError = errMsg
			break
		}
		http.Redirect(w, r, "/auth/register?error="+firstError, http.StatusSeeOther)
		return
	}

	// Send verification email if configured
	if h.emailService.IsConfigured() {
		// Generate verification code and send email
		// For now, auto-verify if email not configured
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"user_id": userID})
		return
	}

	http.Redirect(w, r, "/auth/login?registered=true", http.StatusSeeOther)
}

// Verify handles GET /auth/verify
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	// Email verification would be implemented here
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// PasswordResetPage handles GET /auth/password/reset
func (h *AuthHandler) PasswordResetPage(w http.ResponseWriter, r *http.Request) {
	if !h.emailService.IsConfigured() {
		http.Error(w, "Password reset requires email configuration", http.StatusServiceUnavailable)
		return
	}

	csrfToken := h.securityMW.GenerateCSRFToken()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Password - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; display: flex; justify-content: center; align-items: center; }
        .container { width: 100%; max-width: 400px; padding: 2rem; }
        h1 { text-align: center; margin-bottom: 2rem; color: #bd93f9; }
        form { display: flex; flex-direction: column; gap: 1rem; }
        label { font-size: 0.875rem; color: #6272a4; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #44475a; color: #f8f8f2; font-size: 1rem; }
        button { padding: 0.75rem; border: none; border-radius: 6px; background: #ffb86c; color: #282a36; font-size: 1rem; cursor: pointer; font-weight: 600; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Reset Password</h1>
        <form method="POST" action="/auth/password/reset">
            <input type="hidden" name="csrf_token" value="` + csrfToken + `">
            <div>
                <label for="email">Email</label>
                <input type="email" id="email" name="email" required>
            </div>
            <button type="submit">Send Reset Link</button>
        </form>
    </div>
</body>
</html>`))
}

// PasswordReset handles POST /auth/password/reset
func (h *AuthHandler) PasswordReset(w http.ResponseWriter, r *http.Request) {
	if !h.emailService.IsConfigured() {
		http.Error(w, "Password reset requires email configuration", http.StatusServiceUnavailable)
		return
	}

	// Password reset logic would be implemented here
	// 1. Generate reset token
	// 2. Store token with expiration
	// 3. Send email with reset link

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"reset email sent if account exists"}` + "\n"))
		return
	}

	http.Redirect(w, r, "/auth/login?reset=sent", http.StatusSeeOther)
}

// getClientIP extracts the client IP from a request
func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	xrip := r.Header.Get("X-Real-IP")
	if xrip != "" {
		return xrip
	}
	return r.RemoteAddr
}

// APILogin handles POST /api/v1/auth/login
func (h *AuthHandler) APILogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	// Validate login input (trim identifier, check password whitespace)
	result := service.ValidateLogin(req.Identifier, req.Password)
	if result.HasErrors() {
		SendValidationErrors(w, result.Errors)
		return
	}

	ip := getClientIP(r)

	userID, adminID, err := h.authService.Authenticate(ctx, req.Identifier, req.Password, ip)
	if err != nil {
		switch err {
		case service.ErrAccountLocked:
			SendError(w, ErrAccountLocked, "Account is locked")
		case service.ErrAccountDisabled:
			SendError(w, ErrForbidden, "Account is disabled")
		default:
			SendError(w, ErrUnauthorized, "Invalid credentials")
		}
		return
	}

	sessionID, err := h.authService.CreateSession(ctx, userID, adminID, ip, r.UserAgent())
	if err != nil {
		SendError(w, ErrServerError, "Failed to create session")
		return
	}

	SendOK(w, map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"admin_id":   adminID,
		"is_admin":   adminID != 0,
		"expires_at": time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
	})
}

// APILogout handles POST /api/v1/auth/logout
func (h *AuthHandler) APILogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := middleware.GetSessionID(ctx)

	if sessionID != "" {
		h.authService.InvalidateSession(context.Background(), sessionID)
	}

	SendMessage(w, "Logged out")
}

// APIRegister handles POST /api/v1/auth/register
func (h *AuthHandler) APIRegister(w http.ResponseWriter, r *http.Request) {
	if h.registrationMode == "disabled" {
		SendError(w, ErrNotFound, "Registration is disabled")
		return
	}

	ctx := r.Context()

	var req struct {
		Username        string `json:"username"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	// Validate all registration input
	validationResult := service.ValidateRegistration(req.Username, req.Email, req.Password, req.ConfirmPassword)
	if validationResult.HasErrors() {
		SendValidationErrors(w, validationResult.Errors)
		return
	}

	// Use auth service for registration (includes duplicate checking)
	userID, result, err := h.authService.RegisterUser(ctx, req.Username, req.Email, req.Password, req.ConfirmPassword)
	if err != nil {
		SendError(w, ErrServerError, err.Error())
		return
	}
	if result != nil && result.HasErrors() {
		SendValidationErrors(w, result.Errors)
		return
	}

	SendCreated(w, map[string]interface{}{"user_id": userID})
}
