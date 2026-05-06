// Package swagger - Swagger UI theming
// See AI.md PART 14 for Swagger theming specification
package swagger

// ThemeCSS returns theme-specific CSS for Swagger UI
// See AI.md PART 14 - Dark is default, must support light/dark/auto
func ThemeCSS(theme string) string {
	switch theme {
	case "light":
		return lightThemeCSS
	default:
		return darkThemeCSS
	}
}

// darkThemeCSS is Swagger UI dark theme per AI.md PART 14
// Uses Dracula color palette
var darkThemeCSS = `
/* Swagger UI - Dark Theme */
/* See AI.md PART 14 for color specification */
.swagger-ui.theme-dark {
  background: #282a36;
  color: #f8f8f2;
}

.swagger-ui.theme-dark .topbar {
  background: #1e1f29;
}

.swagger-ui.theme-dark .info .title,
.swagger-ui.theme-dark .opblock-tag {
  color: #f8f8f2;
}

.swagger-ui.theme-dark .opblock.opblock-get {
  background: rgba(139, 233, 253, 0.1);
  border-color: #8be9fd;
}

.swagger-ui.theme-dark .opblock.opblock-get .opblock-summary-method {
  background: #8be9fd;
  color: #282a36;
}

.swagger-ui.theme-dark .opblock.opblock-post {
  background: rgba(80, 250, 123, 0.1);
  border-color: #50fa7b;
}

.swagger-ui.theme-dark .opblock.opblock-post .opblock-summary-method {
  background: #50fa7b;
  color: #282a36;
}

.swagger-ui.theme-dark .opblock.opblock-put {
  background: rgba(255, 184, 108, 0.1);
  border-color: #ffb86c;
}

.swagger-ui.theme-dark .opblock.opblock-put .opblock-summary-method {
  background: #ffb86c;
  color: #282a36;
}

.swagger-ui.theme-dark .opblock.opblock-delete {
  background: rgba(255, 85, 85, 0.1);
  border-color: #ff5555;
}

.swagger-ui.theme-dark .opblock.opblock-delete .opblock-summary-method {
  background: #ff5555;
  color: #282a36;
}

.swagger-ui.theme-dark .opblock.opblock-patch {
  background: rgba(189, 147, 249, 0.1);
  border-color: #bd93f9;
}

.swagger-ui.theme-dark .opblock.opblock-patch .opblock-summary-method {
  background: #bd93f9;
  color: #282a36;
}

.swagger-ui.theme-dark input,
.swagger-ui.theme-dark textarea,
.swagger-ui.theme-dark select {
  background: #44475a;
  color: #f8f8f2;
  border: 1px solid #6272a4;
}

.swagger-ui.theme-dark .btn {
  background: #6272a4;
  color: #f8f8f2;
}

.swagger-ui.theme-dark .btn:hover {
  background: #bd93f9;
}

.swagger-ui.theme-dark .opblock-summary-path,
.swagger-ui.theme-dark .opblock-summary-description {
  color: #f8f8f2;
}

.swagger-ui.theme-dark .parameter__name,
.swagger-ui.theme-dark .parameter__type {
  color: #8be9fd;
}

.swagger-ui.theme-dark .response-col_status {
  color: #50fa7b;
}

.swagger-ui.theme-dark table thead tr th,
.swagger-ui.theme-dark table tbody tr td {
  color: #f8f8f2;
  border-color: #44475a;
}

.swagger-ui.theme-dark .model-title {
  color: #bd93f9;
}

.swagger-ui.theme-dark .prop-type {
  color: #8be9fd;
}

.swagger-ui.theme-dark .model {
  color: #f8f8f2;
}
`

// lightThemeCSS is Swagger UI light theme per AI.md PART 14
var lightThemeCSS = `
/* Swagger UI - Light Theme */
/* See AI.md PART 14 for color specification */
.swagger-ui.theme-light {
  background: #ffffff;
  color: #1a1a1a;
}

.swagger-ui.theme-light .topbar {
  background: #f5f5f5;
  border-bottom: 1px solid #e0e0e0;
}

.swagger-ui.theme-light .info .title,
.swagger-ui.theme-light .opblock-tag {
  color: #1a1a1a;
}

.swagger-ui.theme-light .opblock.opblock-get {
  background: rgba(0, 102, 204, 0.05);
  border-color: #0066cc;
}

.swagger-ui.theme-light .opblock.opblock-get .opblock-summary-method {
  background: #0066cc;
}

.swagger-ui.theme-light .opblock.opblock-post {
  background: rgba(0, 128, 0, 0.05);
  border-color: #008000;
}

.swagger-ui.theme-light .opblock.opblock-post .opblock-summary-method {
  background: #008000;
}

.swagger-ui.theme-light .opblock.opblock-put {
  background: rgba(255, 140, 0, 0.05);
  border-color: #ff8c00;
}

.swagger-ui.theme-light .opblock.opblock-put .opblock-summary-method {
  background: #ff8c00;
}

.swagger-ui.theme-light .opblock.opblock-delete {
  background: rgba(204, 0, 0, 0.05);
  border-color: #cc0000;
}

.swagger-ui.theme-light .opblock.opblock-delete .opblock-summary-method {
  background: #cc0000;
}

.swagger-ui.theme-light .opblock.opblock-patch {
  background: rgba(130, 80, 223, 0.05);
  border-color: #8250df;
}

.swagger-ui.theme-light .opblock.opblock-patch .opblock-summary-method {
  background: #8250df;
}

.swagger-ui.theme-light input,
.swagger-ui.theme-light textarea,
.swagger-ui.theme-light select {
  background: #ffffff;
  color: #1a1a1a;
  border: 1px solid #cccccc;
}

.swagger-ui.theme-light .btn {
  background: #0066cc;
  color: #ffffff;
}

.swagger-ui.theme-light .btn:hover {
  background: #0052a3;
}
`
