package sqlite

import (
	"context"
	"testing"
)

func TestOpenAppliesSchema(t *testing.T) {
	store, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
