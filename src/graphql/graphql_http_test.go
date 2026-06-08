// Package graphql — Tests for HTTP handlers: Handler, PlaygroundHandler, graphiqlHTML,
// writeError. Covers method routing, empty-query rejection, invalid JSON,
// GET with query param, and theme rendering.
package graphql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- writeError ---

func TestWriteErrorSetsStatusCode(t *testing.T) {
	t.Parallel()
	s := NewServer(nil)
	rr := httptest.NewRecorder()
	s.writeError(rr, "something went wrong", http.StatusBadRequest)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("writeError status = %d, want 400", rr.Code)
	}
}

func TestWriteErrorBodyContainsMessage(t *testing.T) {
	t.Parallel()
	s := NewServer(nil)
	rr := httptest.NewRecorder()
	s.writeError(rr, "test error message", http.StatusInternalServerError)

	var resp Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("writeError body not valid JSON: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Fatal("writeError response has no errors")
	}
	if resp.Errors[0].Message != "test error message" {
		t.Errorf("writeError message = %q, want %q", resp.Errors[0].Message, "test error message")
	}
}

// --- Handler (method routing) ---

func TestHandlerPUTMethodReturns405(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodPut, "/graphql", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Handler(PUT) status = %d, want 405", rr.Code)
	}
}

func TestHandlerPOSTEmptyQueryReturns400(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodPost, "/graphql",
		strings.NewReader(`{"query":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Handler(POST, empty query) status = %d, want 400", rr.Code)
	}
}

func TestHandlerPOSTInvalidJSONReturns400(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodPost, "/graphql",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Handler(POST, invalid JSON) status = %d, want 400", rr.Code)
	}
}

func TestHandlerGETNoQueryReturns400(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Handler(GET, no query) status = %d, want 400", rr.Code)
	}
}

func TestHandlerGETWithQueryReturns200(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={tracks}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Handler(GET, query={tracks}) status = %d, want 200", rr.Code)
	}
}

func TestHandlerPOSTQueryReturns200(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodPost, "/graphql",
		strings.NewReader(`{"query":"{ tracks }"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Handler(POST, valid query) status = %d, want 200", rr.Code)
	}
}

func TestHandlerResponseIsJSON(t *testing.T) {
	t.Parallel()
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={tracks}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Handler Content-Type = %q, want application/json", ct)
	}
}

// --- PlaygroundHandler ---

func TestPlaygroundHandlerReturns200(t *testing.T) {
	t.Parallel()
	h := PlaygroundHandler("dark")
	req := httptest.NewRequest(http.MethodGet, "/graphql/playground", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PlaygroundHandler status = %d, want 200", rr.Code)
	}
}

func TestPlaygroundHandlerContentTypeIsHTML(t *testing.T) {
	t.Parallel()
	h := PlaygroundHandler("dark")
	req := httptest.NewRequest(http.MethodGet, "/graphql/playground", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("PlaygroundHandler Content-Type = %q, want text/html", ct)
	}
}

func TestPlaygroundHandlerEmptyThemeDefaultsDark(t *testing.T) {
	t.Parallel()
	h := PlaygroundHandler("")
	req := httptest.NewRequest(http.MethodGet, "/graphql/playground", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PlaygroundHandler(empty theme) status = %d, want 200", rr.Code)
	}
}

// --- graphiqlHTML ---

func TestGraphiqlHTMLContainsDoctype(t *testing.T) {
	t.Parallel()
	html := graphiqlHTML("dark")
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("graphiqlHTML() missing DOCTYPE")
	}
}

func TestGraphiqlHTMLContainsCASRADTitle(t *testing.T) {
	t.Parallel()
	html := graphiqlHTML("dark")
	if !strings.Contains(html, "CASRAD") {
		t.Error("graphiqlHTML() missing CASRAD in title")
	}
}

func TestGraphiqlHTMLLightThemeDiffers(t *testing.T) {
	t.Parallel()
	dark := graphiqlHTML("dark")
	light := graphiqlHTML("light")
	// The themes produce different CSS
	if dark == light {
		t.Error("graphiqlHTML(dark) and graphiqlHTML(light) should differ")
	}
}
