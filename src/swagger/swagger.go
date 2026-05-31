// Package swagger provides OpenAPI/Swagger documentation
package swagger

import (
	"net/http"
)

// swaggerUIHTML is the Swagger UI HTML template.
// All assets are served from /static/swagger/ (embedded at build time per AI.md PART 7).
// No CDN links are used.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CASRAD API Documentation</title>
    <link rel="stylesheet" type="text/css" href="/static/swagger/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="/static/swagger/swagger-ui-bundle.js"></script>
    <script src="/static/swagger/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/api/v1/openapi.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

// openAPISpec is the OpenAPI 3.0 specification
const openAPISpec = `{
  "openapi": "3.0.0",
  "info": {
    "title": "CASRAD API",
    "description": "Complete Audio Streaming, Radio, and Distribution API",
    "version": "1.0.0",
    "contact": {
      "name": "CASRAD Support"
    },
    "license": {
      "name": "MIT"
    }
  },
  "servers": [
    {
      "url": "/api/v1",
      "description": "API v1"
    }
  ],
  "paths": {
    "/auth/login": {
      "post": {
        "summary": "Login",
        "description": "Authenticate with username/email and password",
        "tags": ["Authentication"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/LoginRequest"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Login successful",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/LoginResponse"
                }
              }
            }
          },
          "401": {
            "description": "Invalid credentials"
          },
          "423": {
            "description": "Account locked"
          }
        }
      }
    },
    "/auth/logout": {
      "post": {
        "summary": "Logout",
        "description": "Invalidate current session",
        "tags": ["Authentication"],
        "security": [{"sessionAuth": []}],
        "responses": {
          "200": {
            "description": "Logout successful"
          }
        }
      }
    },
    "/auth/register": {
      "post": {
        "summary": "Register",
        "description": "Create a new user account",
        "tags": ["Authentication"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/RegisterRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Registration successful"
          },
          "400": {
            "description": "Invalid input"
          },
          "409": {
            "description": "Username or email already taken"
          }
        }
      }
    },
    "/users": {
      "get": {
        "summary": "Get current user",
        "description": "Get the currently authenticated user's profile (no ID required - from session)",
        "tags": ["Users"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "responses": {
          "200": {
            "description": "User profile",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/User"
                }
              }
            }
          },
          "401": {
            "description": "Not authenticated"
          }
        }
      },
      "patch": {
        "summary": "Update current user",
        "description": "Update the currently authenticated user's profile (no ID required - from session)",
        "tags": ["Users"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/UserUpdate"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Profile updated"
          },
          "400": {
            "description": "Invalid input"
          }
        }
      }
    },
    "/tracks": {
      "get": {
        "summary": "List tracks",
        "description": "Get a paginated list of tracks",
        "tags": ["Library"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "parameters": [
          {"name": "offset", "in": "query", "schema": {"type": "integer", "default": 0}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 50, "maximum": 100}},
          {"name": "sort", "in": "query", "schema": {"type": "string", "enum": ["title", "artist", "album", "created_at"]}}
        ],
        "responses": {
          "200": {
            "description": "List of tracks",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/TrackList"
                }
              }
            }
          }
        }
      }
    },
    "/tracks/{id}": {
      "get": {
        "summary": "Get track",
        "description": "Get a track by ID",
        "tags": ["Library"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}
        ],
        "responses": {
          "200": {
            "description": "Track details",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Track"
                }
              }
            }
          },
          "404": {
            "description": "Track not found"
          }
        }
      }
    },
    "/tracks/{id}/stream": {
      "get": {
        "summary": "Stream track",
        "description": "Stream a track's audio",
        "tags": ["Streaming"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}},
          {"name": "format", "in": "query", "schema": {"type": "string", "enum": ["mp3", "aac", "opus", "flac", "original"]}},
          {"name": "bitrate", "in": "query", "schema": {"type": "integer", "enum": [64, 128, 192, 256, 320]}}
        ],
        "responses": {
          "200": {
            "description": "Audio stream",
            "content": {
              "audio/mpeg": {},
              "audio/aac": {},
              "audio/ogg": {},
              "audio/flac": {}
            }
          },
          "404": {
            "description": "Track not found"
          }
        }
      }
    },
    "/playlists": {
      "get": {
        "summary": "List playlists",
        "description": "Get user's playlists",
        "tags": ["Playlists"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "responses": {
          "200": {
            "description": "List of playlists",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/PlaylistList"
                }
              }
            }
          }
        }
      },
      "post": {
        "summary": "Create playlist",
        "description": "Create a new playlist",
        "tags": ["Playlists"],
        "security": [{"sessionAuth": []}, {"apiToken": []}],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/PlaylistCreate"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Playlist created"
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "sessionAuth": {
        "type": "apiKey",
        "in": "cookie",
        "name": "session"
      },
      "apiToken": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "API Token"
      }
    },
    "schemas": {
      "LoginRequest": {
        "type": "object",
        "required": ["identifier", "password"],
        "properties": {
          "identifier": {"type": "string", "description": "Username, email, or user ID"},
          "password": {"type": "string", "format": "password"}
        }
      },
      "LoginResponse": {
        "type": "object",
        "properties": {
          "session_id": {"type": "string"},
          "user_id": {"type": "integer"},
          "is_admin": {"type": "boolean"},
          "expires_at": {"type": "string", "format": "date-time"}
        }
      },
      "RegisterRequest": {
        "type": "object",
        "required": ["username", "email", "password"],
        "properties": {
          "username": {"type": "string", "minLength": 3, "maxLength": 32, "pattern": "^[a-z][a-z0-9_-]{2,31}$"},
          "email": {"type": "string", "format": "email"},
          "password": {"type": "string", "format": "password", "minLength": 8}
        }
      },
      "User": {
        "type": "object",
        "properties": {
          "id": {"type": "integer"},
          "username": {"type": "string"},
          "email": {"type": "string"},
          "role": {"type": "string"},
          "theme_preference": {"type": "string"},
          "storage_quota_bytes": {"type": "integer"},
          "storage_used_bytes": {"type": "integer"},
          "email_verified": {"type": "boolean"},
          "created_at": {"type": "string", "format": "date-time"}
        }
      },
      "UserUpdate": {
        "type": "object",
        "properties": {
          "theme_preference": {"type": "string", "enum": ["dark", "light", "auto"]},
          "bio": {"type": "string", "maxLength": 500},
          "website": {"type": "string", "format": "uri"},
          "location": {"type": "string", "maxLength": 100}
        }
      },
      "Track": {
        "type": "object",
        "properties": {
          "id": {"type": "integer"},
          "title": {"type": "string"},
          "artist": {"type": "string"},
          "album": {"type": "string"},
          "duration": {"type": "integer", "description": "Duration in milliseconds"},
          "bitrate": {"type": "integer"},
          "file_type": {"type": "string"},
          "play_count": {"type": "integer"},
          "rating": {"type": "integer", "minimum": 0, "maximum": 5}
        }
      },
      "TrackList": {
        "type": "object",
        "properties": {
          "tracks": {"type": "array", "items": {"$ref": "#/components/schemas/Track"}},
          "total": {"type": "integer"},
          "offset": {"type": "integer"},
          "limit": {"type": "integer"}
        }
      },
      "PlaylistCreate": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name": {"type": "string"},
          "description": {"type": "string"},
          "is_public": {"type": "boolean", "default": false}
        }
      },
      "PlaylistList": {
        "type": "object",
        "properties": {
          "playlists": {"type": "array", "items": {"$ref": "#/components/schemas/Playlist"}},
          "total": {"type": "integer"}
        }
      },
      "Playlist": {
        "type": "object",
        "properties": {
          "id": {"type": "integer"},
          "name": {"type": "string"},
          "description": {"type": "string"},
          "is_public": {"type": "boolean"},
          "track_count": {"type": "integer"},
          "duration_ms": {"type": "integer"},
          "created_at": {"type": "string", "format": "date-time"}
        }
      }
    }
  }
}`

// Handler returns an HTTP handler for Swagger UI
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerUIHTML))
	})
}

// Spec returns the OpenAPI specification
// Per AI.md PART 14: JSON must end with single trailing newline
func Spec() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(openAPISpec))
		w.Write([]byte("\n"))
	})
}
