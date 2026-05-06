// Package graphql - GraphiQL theming
// See AI.md PART 14 for GraphiQL theming specification
package graphql

// ThemeCSS returns theme-specific CSS for GraphiQL
// See AI.md PART 14 - Dark is default, must support light/dark/auto
func ThemeCSS(theme string) string {
	switch theme {
	case "light":
		return lightThemeCSS
	default:
		return darkThemeCSS
	}
}

// darkThemeCSS is GraphiQL dark theme per AI.md PART 14
// Uses Dracula color palette
var darkThemeCSS = `
/* GraphiQL - Dark Theme */
/* See AI.md PART 14 for color specification */
.graphiql-container.theme-dark {
  background: #282a36;
  color: #f8f8f2;
}

.graphiql-container.theme-dark .CodeMirror {
  background: #282a36;
  color: #f8f8f2;
}

.graphiql-container.theme-dark .CodeMirror-gutters {
  background: #1e1f29;
  border-right: 1px solid #44475a;
}

.graphiql-container.theme-dark .result-window {
  background: #282a36;
}

.graphiql-container.theme-dark .execute-button {
  background: #50fa7b;
  color: #282a36;
}

.graphiql-container.theme-dark .execute-button:hover {
  background: #69ff94;
}

.graphiql-container.theme-dark .toolbar-button {
  background: #44475a;
  color: #f8f8f2;
}

.graphiql-container.theme-dark .toolbar-button:hover {
  background: #6272a4;
}

.graphiql-container.theme-dark .doc-explorer-title {
  color: #bd93f9;
}

.graphiql-container.theme-dark .doc-explorer-contents {
  background: #282a36;
  color: #f8f8f2;
}

.graphiql-container.theme-dark .type-name {
  color: #8be9fd;
}

.graphiql-container.theme-dark .field-name {
  color: #50fa7b;
}

.graphiql-container.theme-dark .arg-name {
  color: #ffb86c;
}

.graphiql-container.theme-dark .keyword {
  color: #ff79c6;
}

.graphiql-container.theme-dark .cm-comment {
  color: #6272a4;
}

.graphiql-container.theme-dark .cm-string {
  color: #f1fa8c;
}

.graphiql-container.theme-dark .cm-number {
  color: #bd93f9;
}

.graphiql-container.theme-dark .cm-atom {
  color: #bd93f9;
}
`

// lightThemeCSS is GraphiQL light theme per AI.md PART 14
var lightThemeCSS = `
/* GraphiQL - Light Theme */
/* See AI.md PART 14 for color specification */
.graphiql-container.theme-light {
  background: #ffffff;
  color: #1a1a1a;
}

.graphiql-container.theme-light .CodeMirror {
  background: #ffffff;
  color: #1a1a1a;
}

.graphiql-container.theme-light .CodeMirror-gutters {
  background: #f5f5f5;
  border-right: 1px solid #e0e0e0;
}

.graphiql-container.theme-light .result-window {
  background: #ffffff;
}

.graphiql-container.theme-light .execute-button {
  background: #008000;
  color: #ffffff;
}

.graphiql-container.theme-light .execute-button:hover {
  background: #006600;
}

.graphiql-container.theme-light .toolbar-button {
  background: #f5f5f5;
  color: #1a1a1a;
  border: 1px solid #cccccc;
}

.graphiql-container.theme-light .toolbar-button:hover {
  background: #e0e0e0;
}

.graphiql-container.theme-light .doc-explorer-title {
  color: #8250df;
}

.graphiql-container.theme-light .type-name {
  color: #0066cc;
}

.graphiql-container.theme-light .field-name {
  color: #008000;
}

.graphiql-container.theme-light .arg-name {
  color: #ff8c00;
}
`
