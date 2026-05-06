// Package graphql provides GraphQL API
// See AI.md - GraphQL is REQUIRED, always at src/graphql/
package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/store"
)

// Server represents a GraphQL server
type Server struct {
	store    store.Store
	resolver *Resolver
}

// NewServer creates a new GraphQL server
func NewServer(s store.Store) *Server {
	return &Server{
		store:    s,
		resolver: NewResolver(s),
	}
}

// Request represents a GraphQL request
type Request struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// Response represents a GraphQL response
type Response struct {
	Data   interface{}     `json:"data,omitempty"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message   string                 `json:"message"`
	Locations []ErrorLocation        `json:"locations,omitempty"`
	Path      []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// ErrorLocation represents error location in query
type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Handler returns an HTTP handler for GraphQL endpoint
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req Request

		switch r.Method {
		case http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				s.writeError(w, "Invalid JSON request body", http.StatusBadRequest)
				return
			}
		case http.MethodGet:
			req.Query = r.URL.Query().Get("query")
			req.OperationName = r.URL.Query().Get("operationName")
			if vars := r.URL.Query().Get("variables"); vars != "" {
				json.Unmarshal([]byte(vars), &req.Variables)
			}
		default:
			s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if req.Query == "" {
			s.writeError(w, "Query is required", http.StatusBadRequest)
			return
		}

		// Build context with user info
		ctx := r.Context()

		// Execute query
		resp := s.execute(ctx, &req)

		json.NewEncoder(w).Encode(resp)
	})
}

// execute executes a GraphQL query
func (s *Server) execute(ctx context.Context, req *Request) *Response {
	query := strings.TrimSpace(req.Query)

	// Detect if it's a mutation or query
	if strings.HasPrefix(query, "mutation") {
		return s.executeMutation(ctx, query, req.Variables)
	}

	return s.executeQuery(ctx, query, req.Variables)
}

// executeQuery executes a GraphQL query
func (s *Server) executeQuery(ctx context.Context, query string, variables map[string]interface{}) *Response {
	data := make(map[string]interface{})
	var errors []GraphQLError

	// Parse the query to extract fields
	fields := parseQueryFields(query)

	for _, field := range fields {
		result, err := s.resolveQueryField(ctx, field, variables)
		if err != nil {
			errors = append(errors, GraphQLError{Message: err.Error()})
		} else {
			data[field.Name] = result
		}
	}

	return &Response{Data: data, Errors: errors}
}

// executeMutation executes a GraphQL mutation
func (s *Server) executeMutation(ctx context.Context, query string, variables map[string]interface{}) *Response {
	data := make(map[string]interface{})
	var errors []GraphQLError

	// Parse mutation fields
	fields := parseMutationFields(query)

	for _, field := range fields {
		result, err := s.resolveMutationField(ctx, field, variables)
		if err != nil {
			errors = append(errors, GraphQLError{Message: err.Error()})
		} else {
			data[field.Name] = result
		}
	}

	return &Response{Data: data, Errors: errors}
}

// Field represents a parsed field
type Field struct {
	Name      string
	Arguments map[string]interface{}
	SubFields []string
}

// parseQueryFields extracts field names from a query
func parseQueryFields(query string) []Field {
	var fields []Field

	// Simple parser - look for known query fields
	queryFields := []string{"health", "me", "tracks", "track", "albums", "album", "artists", "artist", "playlists", "playlist", "broadcasts", "broadcast", "search"}

	for _, fieldName := range queryFields {
		if strings.Contains(query, fieldName) {
			field := Field{
				Name:      fieldName,
				Arguments: make(map[string]interface{}),
			}

			// Extract arguments if present (e.g., track(id: "123"))
			start := strings.Index(query, fieldName+"(")
			if start != -1 {
				end := strings.Index(query[start:], ")")
				if end != -1 {
					argsStr := query[start+len(fieldName)+1 : start+end]
					field.Arguments = parseArguments(argsStr)
				}
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// parseMutationFields extracts field names from a mutation
func parseMutationFields(query string) []Field {
	var fields []Field

	mutationFields := []string{"login", "logout", "createPlaylist", "updatePlaylist", "deletePlaylist", "addToPlaylist", "updateProfile"}

	for _, fieldName := range mutationFields {
		if strings.Contains(query, fieldName) {
			field := Field{
				Name:      fieldName,
				Arguments: make(map[string]interface{}),
			}

			// Extract arguments
			start := strings.Index(query, fieldName+"(")
			if start != -1 {
				end := findMatchingParen(query[start+len(fieldName):])
				if end != -1 {
					argsStr := query[start+len(fieldName)+1 : start+len(fieldName)+end]
					field.Arguments = parseArguments(argsStr)
				}
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// findMatchingParen finds the index of the matching closing parenthesis
func findMatchingParen(s string) int {
	depth := 1
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseArguments parses GraphQL arguments string
func parseArguments(argsStr string) map[string]interface{} {
	args := make(map[string]interface{})

	// Simple key:value parsing
	argsStr = strings.TrimSpace(argsStr)
	if argsStr == "" {
		return args
	}

	// Split by commas (not inside strings)
	parts := splitArguments(argsStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:colonIdx])
		value := strings.TrimSpace(part[colonIdx+1:])

		// Parse value
		args[key] = parseValue(value)
	}

	return args
}

// splitArguments splits arguments by comma, handling nested structures
func splitArguments(s string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	inString := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '"' && (i == 0 || s[i-1] != '\\') {
			inString = !inString
		}

		if !inString {
			if c == '{' || c == '[' {
				depth++
			} else if c == '}' || c == ']' {
				depth--
			} else if c == ',' && depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
		}

		current.WriteByte(c)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseValue parses a GraphQL value
func parseValue(s string) interface{} {
	s = strings.TrimSpace(s)

	// String
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}

	// Number
	if n := parseNumber(s); n != nil {
		return n
	}

	// Boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Null
	if s == "null" {
		return nil
	}

	// Array
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return parseArray(s[1 : len(s)-1])
	}

	// Object
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return parseArguments(s[1 : len(s)-1])
	}

	return s
}

// parseNumber attempts to parse a number
func parseNumber(s string) interface{} {
	// Try int
	var i int64
	if err := json.Unmarshal([]byte(s), &i); err == nil {
		return i
	}

	// Try float
	var f float64
	if err := json.Unmarshal([]byte(s), &f); err == nil {
		return f
	}

	return nil
}

// parseArray parses a GraphQL array
func parseArray(s string) []interface{} {
	var arr []interface{}
	parts := splitArguments(s)
	for _, part := range parts {
		arr = append(arr, parseValue(strings.TrimSpace(part)))
	}
	return arr
}

// resolveQueryField resolves a single query field
func (s *Server) resolveQueryField(ctx context.Context, field Field, variables map[string]interface{}) (interface{}, error) {
	switch field.Name {
	case "health":
		return s.resolver.Health(ctx)
	case "me":
		return s.resolver.Me(ctx)
	case "tracks":
		offset, _ := field.Arguments["offset"].(int64)
		limit, _ := field.Arguments["limit"].(int64)
		if limit == 0 {
			limit = 50
		}
		return s.resolver.Tracks(ctx, int(offset), int(limit))
	case "track":
		id, _ := field.Arguments["id"].(string)
		return s.resolver.Track(ctx, id)
	case "albums":
		offset, _ := field.Arguments["offset"].(int64)
		limit, _ := field.Arguments["limit"].(int64)
		if limit == 0 {
			limit = 50
		}
		return s.resolver.Albums(ctx, int(offset), int(limit))
	case "album":
		id, _ := field.Arguments["id"].(string)
		return s.resolver.Album(ctx, id)
	case "artists":
		offset, _ := field.Arguments["offset"].(int64)
		limit, _ := field.Arguments["limit"].(int64)
		if limit == 0 {
			limit = 50
		}
		return s.resolver.Artists(ctx, int(offset), int(limit))
	case "artist":
		id, _ := field.Arguments["id"].(string)
		return s.resolver.Artist(ctx, id)
	case "playlists":
		return s.resolver.Playlists(ctx)
	case "playlist":
		id, _ := field.Arguments["id"].(string)
		return s.resolver.Playlist(ctx, id)
	case "broadcasts":
		return s.resolver.Broadcasts(ctx)
	case "broadcast":
		id, _ := field.Arguments["id"].(string)
		return s.resolver.Broadcast(ctx, id)
	case "search":
		query, _ := field.Arguments["query"].(string)
		return s.resolver.Search(ctx, query)
	default:
		return nil, nil
	}
}

// resolveMutationField resolves a single mutation field
func (s *Server) resolveMutationField(ctx context.Context, field Field, variables map[string]interface{}) (interface{}, error) {
	switch field.Name {
	case "login":
		identifier, _ := field.Arguments["identifier"].(string)
		password, _ := field.Arguments["password"].(string)
		return s.resolver.Login(ctx, identifier, password)
	case "logout":
		return s.resolver.Logout(ctx)
	case "createPlaylist":
		input, _ := field.Arguments["input"].(map[string]interface{})
		return s.resolver.CreatePlaylist(ctx, input)
	case "updatePlaylist":
		id, _ := field.Arguments["id"].(string)
		input, _ := field.Arguments["input"].(map[string]interface{})
		return s.resolver.UpdatePlaylist(ctx, id, input)
	case "deletePlaylist":
		id, _ := field.Arguments["id"].(string)
		return s.resolver.DeletePlaylist(ctx, id)
	case "addToPlaylist":
		playlistID, _ := field.Arguments["playlistId"].(string)
		trackIDs, _ := field.Arguments["trackIds"].([]interface{})
		var ids []string
		for _, id := range trackIDs {
			if s, ok := id.(string); ok {
				ids = append(ids, s)
			}
		}
		return s.resolver.AddToPlaylist(ctx, playlistID, ids)
	case "updateProfile":
		input, _ := field.Arguments["input"].(map[string]interface{})
		return s.resolver.UpdateProfile(ctx, input)
	default:
		return nil, nil
	}
}

// writeError writes a GraphQL error response
func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Errors: []GraphQLError{{Message: message}},
	})
}

// Handler returns an HTTP handler for GraphQL endpoint (standalone function)
func Handler(s store.Store) http.Handler {
	srv := NewServer(s)
	return srv.Handler()
}

// PlaygroundHandler returns an HTTP handler for GraphiQL
func PlaygroundHandler(theme string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Check for user theme preference
		if theme == "" {
			theme = "dark"
			if userID := middleware.GetUserID(r.Context()); userID > 0 {
				// Could look up user preference here
			}
		}

		// Render GraphiQL with theme
		html := graphiqlHTML(theme)
		w.Write([]byte(html))
	})
}

// graphiqlHTML returns the GraphiQL playground HTML
func graphiqlHTML(theme string) string {
	themeCSS := ThemeCSS(theme)

	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GraphQL Playground - CASRAD</title>
    <link rel="stylesheet" href="https://unpkg.com/graphiql@3.0.6/graphiql.min.css" />
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        html, body { height: 100%; }
        body {
            font-family: 'Inter', system-ui, sans-serif;
        }
        #graphiql { height: 100vh; }
        ` + themeCSS + `
    </style>
</head>
<body>
    <div id="graphiql"></div>
    <script crossorigin src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/graphiql@3.0.6/graphiql.min.js"></script>
    <script>
        const fetcher = GraphiQL.createFetcher({ url: '/graphql' });
        ReactDOM.createRoot(document.getElementById('graphiql')).render(
            React.createElement(GraphiQL, {
                fetcher: fetcher,
                defaultEditorToolsVisibility: true,
            })
        );
    </script>
</body>
</html>`
}
