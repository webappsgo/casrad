// Package middleware — Tests for Logging and Recovery middleware.
// Covers: responseWriter (WriteHeader captures code, Write captures bytes),
// Logging (calls next, captures method/path), Recovery (handles panics with 500).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- responseWriter ---

func TestResponseWriterCapturesStatusCode(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want 201", rw.statusCode)
	}
}

func TestResponseWriterCapturesByteCount(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}
	n, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Write returned %d bytes, want 5", n)
	}
	if rw.bytes != 5 {
		t.Errorf("bytes = %d, want 5", rw.bytes)
	}
}

// --- Logging middleware ---

func TestLoggingCallsNext(t *testing.T) {
	t.Parallel()
	called := false
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("Logging middleware should call next handler")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

// --- Recovery middleware ---

func TestRecoveryHandlesPanic(t *testing.T) {
	t.Parallel()
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("panic recovery status = %d, want 500", rr.Code)
	}
}

func TestRecoveryNoPanicPassesThrough(t *testing.T) {
	t.Parallel()
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Errorf("no-panic status = %d, want 202", rr.Code)
	}
}
