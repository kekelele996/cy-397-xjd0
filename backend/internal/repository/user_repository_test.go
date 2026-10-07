package repository

import (
	"errors"
	"testing"

	"github.com/contractapi/contractapi/internal/model"
)

func TestUserRepository(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))

	user := &model.User{Username: "alice", PasswordHash: "hash"}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.ID == 0 {
		t.Fatal("Create() should assign id")
	}

	dup := &model.User{Username: "alice", PasswordHash: "hash2"}
	if err := repo.Create(dup); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("Create(duplicate) error = %v, want ErrDuplicateKey", err)
	}

	got, err := repo.FindByUsername("alice")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("FindByUsername() id = %d, want %d", got.ID, user.ID)
	}

	if _, err := repo.FindByUsername("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindByUsername(missing) error = %v, want ErrNotFound", err)
	}

	got, err = repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if got.Username != "alice" {
		t.Fatalf("FindByID() username = %q", got.Username)
	}
}
