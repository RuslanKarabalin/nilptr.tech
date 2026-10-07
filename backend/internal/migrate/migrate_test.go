package migrate

import (
	"os"
	"testing"
)

func TestList(t *testing.T) {
	ms, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 || ms[0].Version != "0001_init" {
		t.Fatalf("migrations %+v", ms)
	}
	for i := 1; i < len(ms); i++ {
		if ms[i-1].Version >= ms[i].Version {
			t.Fatal("migrations not sorted")
		}
	}
}

// TestInitMatchesDocs keeps docs/db.sql identical to the first migration.
func TestInitMatchesDocs(t *testing.T) {
	doc, err := os.ReadFile("../../../docs/db.sql")
	if os.IsNotExist(err) {
		t.Skip("docs/db.sql not available")
	}
	if err != nil {
		t.Fatal(err)
	}
	ms, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if ms[0].SQL != string(doc) {
		t.Fatal("internal/migrate/migrations/0001_init.sql differs from docs/db.sql")
	}
}
