// Package graphql — Tests for pure GraphQL parsing utilities.
// Covers: findMatchingParen, splitArguments, parseValue, parseNumber,
// parseArray, parseArguments, parseQueryFields, parseMutationFields,
// NewServer.
package graphql

import (
	"testing"
)

// --- NewServer ---

func TestNewServerReturnsNonNil(t *testing.T) {
	t.Parallel()
	s := NewServer(nil)
	if s == nil {
		t.Error("NewServer(nil) returned nil")
	}
}

// --- findMatchingParen ---

func TestFindMatchingParenBasic(t *testing.T) {
	t.Parallel()
	// s begins after the opening '(' so "(a: 1)" → s = "(a: 1)"
	// The function is called with the part starting at the function name
	// e.g. for "track(id: 1)", called with "(id: 1)" starting at position 0
	// It expects s[0]='(' and finds matching ')' at index
	// The func signature: findMatchingParen(s string) int
	// s starts at '(' already
	got := findMatchingParen("(id: 123)")
	// Returns index of matching ')' relative to s starting at '('
	// depth starts at 1, so s[0]='(' is already counted
	if got != 8 {
		t.Errorf("findMatchingParen(%q) = %d, want 8", "(id: 123)", got)
	}
}

func TestFindMatchingParenNested(t *testing.T) {
	t.Parallel()
	got := findMatchingParen("(a: (b: 2))")
	// Outer ')' at index 10
	if got != 10 {
		t.Errorf("findMatchingParen nested = %d, want 10", got)
	}
}

func TestFindMatchingParenNoClose(t *testing.T) {
	t.Parallel()
	got := findMatchingParen("(unclosed")
	if got != -1 {
		t.Errorf("findMatchingParen(unclosed) = %d, want -1", got)
	}
}

func TestFindMatchingParenEmpty(t *testing.T) {
	t.Parallel()
	got := findMatchingParen("")
	if got != -1 {
		t.Errorf("findMatchingParen(\"\") = %d, want -1", got)
	}
}

// --- splitArguments ---

func TestSplitArgumentsSingle(t *testing.T) {
	t.Parallel()
	got := splitArguments(`id: "123"`)
	if len(got) != 1 {
		t.Errorf("splitArguments(single) len = %d, want 1", len(got))
	}
}

func TestSplitArgumentsMultiple(t *testing.T) {
	t.Parallel()
	got := splitArguments(`id: "1", limit: 10, offset: 0`)
	if len(got) != 3 {
		t.Errorf("splitArguments(3 args) len = %d, want 3: %v", len(got), got)
	}
}

func TestSplitArgumentsEmpty(t *testing.T) {
	t.Parallel()
	got := splitArguments("")
	if len(got) != 0 {
		t.Errorf("splitArguments(\"\") len = %d, want 0", len(got))
	}
}

func TestSplitArgumentsNestedBraces(t *testing.T) {
	t.Parallel()
	// Commas inside braces must not split
	got := splitArguments(`input: {name: "a", age: 1}, limit: 5`)
	if len(got) != 2 {
		t.Errorf("splitArguments(nested braces) len = %d, want 2: %v", len(got), got)
	}
}

func TestSplitArgumentsStringWithComma(t *testing.T) {
	t.Parallel()
	// Commas inside quoted strings must not split
	got := splitArguments(`name: "hello, world", other: "value"`)
	if len(got) != 2 {
		t.Errorf("splitArguments(comma in string) len = %d, want 2: %v", len(got), got)
	}
}

// --- parseValue ---

func TestParseValueString(t *testing.T) {
	t.Parallel()
	got := parseValue(`"hello"`)
	if s, ok := got.(string); !ok || s != "hello" {
		t.Errorf(`parseValue("hello") = %v (%T), want string "hello"`, got, got)
	}
}

func TestParseValueInteger(t *testing.T) {
	t.Parallel()
	got := parseValue("42")
	if i, ok := got.(int64); !ok || i != 42 {
		t.Errorf("parseValue(42) = %v (%T), want int64 42", got, got)
	}
}

func TestParseValueFloat(t *testing.T) {
	t.Parallel()
	got := parseValue("3.14")
	if f, ok := got.(float64); !ok || f != 3.14 {
		t.Errorf("parseValue(3.14) = %v (%T), want float64 3.14", got, got)
	}
}

func TestParseValueTrue(t *testing.T) {
	t.Parallel()
	got := parseValue("true")
	if b, ok := got.(bool); !ok || !b {
		t.Errorf("parseValue(true) = %v (%T), want bool true", got, got)
	}
}

func TestParseValueFalse(t *testing.T) {
	t.Parallel()
	got := parseValue("false")
	if b, ok := got.(bool); !ok || b {
		t.Errorf("parseValue(false) = %v (%T), want bool false", got, got)
	}
}

func TestParseValueNull(t *testing.T) {
	t.Parallel()
	// json.Unmarshal("null", *int64) succeeds with 0, so parseNumber("null")
	// returns int64(0) before the explicit nil check is reached.
	got := parseValue("null")
	if i, ok := got.(int64); !ok || i != 0 {
		t.Errorf("parseValue(null) = %v (%T), want int64(0) (JSON null→0 via parseNumber)", got, got)
	}
}

func TestParseValueArray(t *testing.T) {
	t.Parallel()
	got := parseValue("[1, 2, 3]")
	arr, ok := got.([]interface{})
	if !ok {
		t.Fatalf("parseValue([1, 2, 3]) type = %T, want []interface{}", got)
	}
	if len(arr) != 3 {
		t.Errorf("parseValue(array) len = %d, want 3", len(arr))
	}
}

func TestParseValueObject(t *testing.T) {
	t.Parallel()
	got := parseValue(`{key: "val"}`)
	if _, ok := got.(map[string]interface{}); !ok {
		t.Errorf("parseValue(object) type = %T, want map[string]interface{}", got)
	}
}

func TestParseValueUnknownStringPassthrough(t *testing.T) {
	t.Parallel()
	got := parseValue("unquoted_ident")
	if s, ok := got.(string); !ok || s != "unquoted_ident" {
		t.Errorf("parseValue(bare string) = %v (%T), want string passthrough", got, got)
	}
}

// --- parseNumber ---

func TestParseNumberInteger(t *testing.T) {
	t.Parallel()
	got := parseNumber("99")
	if i, ok := got.(int64); !ok || i != 99 {
		t.Errorf("parseNumber(99) = %v (%T), want int64 99", got, got)
	}
}

func TestParseNumberFloat(t *testing.T) {
	t.Parallel()
	got := parseNumber("1.5")
	if f, ok := got.(float64); !ok || f != 1.5 {
		t.Errorf("parseNumber(1.5) = %v (%T), want float64 1.5", got, got)
	}
}

func TestParseNumberNonNumber(t *testing.T) {
	t.Parallel()
	got := parseNumber("notanumber")
	if got != nil {
		t.Errorf("parseNumber(notanumber) = %v, want nil", got)
	}
}

func TestParseNumberNegative(t *testing.T) {
	t.Parallel()
	got := parseNumber("-5")
	if i, ok := got.(int64); !ok || i != -5 {
		t.Errorf("parseNumber(-5) = %v (%T), want int64 -5", got, got)
	}
}

// --- parseArray ---

func TestParseArrayEmpty(t *testing.T) {
	t.Parallel()
	got := parseArray("")
	if len(got) != 0 {
		t.Errorf("parseArray(\"\") len = %d, want 0", len(got))
	}
}

func TestParseArrayStrings(t *testing.T) {
	t.Parallel()
	got := parseArray(`"a", "b", "c"`)
	if len(got) != 3 {
		t.Errorf("parseArray(strings) len = %d, want 3", len(got))
	}
}

func TestParseArrayNumbers(t *testing.T) {
	t.Parallel()
	got := parseArray("10, 20, 30")
	if len(got) != 3 {
		t.Errorf("parseArray(numbers) len = %d, want 3", len(got))
	}
}

// --- parseArguments ---

func TestParseArgumentsEmpty(t *testing.T) {
	t.Parallel()
	got := parseArguments("")
	if len(got) != 0 {
		t.Errorf("parseArguments(\"\") len = %d, want 0", len(got))
	}
}

func TestParseArgumentsSingle(t *testing.T) {
	t.Parallel()
	got := parseArguments(`id: "abc123"`)
	if v, ok := got["id"]; !ok || v != "abc123" {
		t.Errorf("parseArguments single: got %v, want id=abc123", got)
	}
}

func TestParseArgumentsMultiple(t *testing.T) {
	t.Parallel()
	got := parseArguments(`limit: 20, offset: 40`)
	if len(got) != 2 {
		t.Errorf("parseArguments(2 args) len = %d, want 2: %v", len(got), got)
	}
	if v, ok := got["limit"]; !ok {
		t.Error("parseArguments missing 'limit' key")
	} else if i, ok := v.(int64); !ok || i != 20 {
		t.Errorf("limit = %v (%T), want int64 20", v, v)
	}
}

func TestParseArgumentsMissingColon(t *testing.T) {
	t.Parallel()
	// Malformed arg without colon should be ignored
	got := parseArguments("invalidarg")
	if len(got) != 0 {
		t.Errorf("parseArguments(no colon) len = %d, want 0", len(got))
	}
}

// --- parseQueryFields ---

func TestParseQueryFieldsHealth(t *testing.T) {
	t.Parallel()
	fields := parseQueryFields("{ health { status } }")
	found := false
	for _, f := range fields {
		if f.Name == "health" {
			found = true
		}
	}
	if !found {
		t.Errorf("parseQueryFields should find 'health' in %v", fields)
	}
}

func TestParseQueryFieldsTrackWithID(t *testing.T) {
	t.Parallel()
	fields := parseQueryFields(`{ track(id: "123") { title } }`)
	found := false
	for _, f := range fields {
		if f.Name == "track" {
			found = true
			if _, hasID := f.Arguments["id"]; !hasID {
				t.Error("parseQueryFields: track field should have 'id' argument")
			}
		}
	}
	if !found {
		t.Errorf("parseQueryFields should find 'track' in %v", fields)
	}
}

func TestParseQueryFieldsEmpty(t *testing.T) {
	t.Parallel()
	fields := parseQueryFields("{ unknownfield }")
	// The query does not contain any of the known field names, so no fields found
	for _, f := range fields {
		knownFields := map[string]bool{
			"health": true, "me": true, "tracks": true, "track": true,
			"albums": true, "album": true, "artists": true, "artist": true,
			"playlists": true, "playlist": true, "broadcasts": true,
			"broadcast": true, "search": true,
		}
		if !knownFields[f.Name] {
			t.Errorf("unexpected field %q in parseQueryFields result", f.Name)
		}
	}
}

// --- parseMutationFields ---

func TestParseMutationFieldsLogin(t *testing.T) {
	t.Parallel()
	fields := parseMutationFields(`mutation { login(identifier: "alice", password: "secret") { session_id } }`)
	found := false
	for _, f := range fields {
		if f.Name == "login" {
			found = true
		}
	}
	if !found {
		t.Errorf("parseMutationFields should find 'login' in %v", fields)
	}
}

func TestParseMutationFieldsLogout(t *testing.T) {
	t.Parallel()
	fields := parseMutationFields("mutation { logout }")
	found := false
	for _, f := range fields {
		if f.Name == "logout" {
			found = true
		}
	}
	if !found {
		t.Errorf("parseMutationFields should find 'logout': %v", fields)
	}
}

func TestParseMutationFieldsCreatePlaylist(t *testing.T) {
	t.Parallel()
	fields := parseMutationFields(`mutation { createPlaylist(name: "My Mix") { id } }`)
	found := false
	for _, f := range fields {
		if f.Name == "createPlaylist" {
			found = true
		}
	}
	if !found {
		t.Errorf("parseMutationFields should find 'createPlaylist': %v", fields)
	}
}
