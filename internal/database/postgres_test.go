package database

import (
	"context"
	"testing"
)

func TestNewPoolRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	pool, err := NewPool(context.Background(), Config{})

	if err == nil {
		t.Fatal("NewPool() error = nil, desired an error for an empty URL")
	}

	if pool != nil {
		t.Fatal("NewPool() pool is not nil, desired nil when configuration is invalid")
	}
}
