package migration

import "testing"

func TestSafeIdentifier(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"schema_migrations": true,
		"SchemaMigrations":  true,
		"_private":          true,
		"schema1":           true,
		"":                  false,
		"1schema":           false,
		"schema-name":       false,
		"schema.name":       false,
	}
	for input, want := range tests {
		if got := safeIdentifier(input); got != want {
			t.Fatalf("safeIdentifier(%q) = %v, want %v", input, got, want)
		}
	}
}
