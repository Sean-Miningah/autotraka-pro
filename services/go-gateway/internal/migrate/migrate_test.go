package migrate

import (
	"embed"
	"strings"
	"testing"
)

func TestMigrationsFSContainsFiles(t *testing.T) {
	got := migrationsFS
	if got == (embed.FS{}) {
		t.Fatal("migrationsFS is empty — no migration files embedded")
	}

	entries, err := got.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no migration files found in embedded filesystem")
	}

	hasUp := false
	hasDown := false
	for _, e := range entries {
		if !e.IsDir() {
			name := e.Name()
			if strings.HasSuffix(name, ".up.sql") {
				hasUp = true
			}
			if strings.HasSuffix(name, ".down.sql") {
				hasDown = true
			}
		}
	}

	if !hasUp {
		t.Error("no .up.sql migration files found")
	}
	if !hasDown {
		t.Error("no .down.sql migration files found")
	}
}