package idgen

import "testing"

func TestNewInvalidNodeID(t *testing.T) {
	_, err := New(-1)
	if err == nil {
		t.Fatal("expected error for negative nodeID")
	}
	_, err = New(1024)
	if err == nil {
		t.Fatal("expected error for nodeID > 1023")
	}
}

func TestGenerateUniqueness(t *testing.T) {
	g, err := New(1)
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[int64]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := g.Generate()
		if id <= 0 {
			t.Fatalf("expected positive ID, got %d", id)
		}
		if seen[id] {
			t.Fatalf("duplicate ID: %d", id)
		}
		seen[id] = true
	}
}
