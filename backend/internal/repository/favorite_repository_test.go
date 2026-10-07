package repository

import (
	"errors"
	"testing"

	"github.com/contractapi/contractapi/internal/model"
)

func TestFavoriteRepository(t *testing.T) {
	db := newTestDB(t)
	templateRepo := NewTemplateRepository(db)
	favRepo := NewFavoriteRepository(db)

	tmpl := &model.ContractTemplate{Code: "lease", Name: "房屋租赁合同", Category: "lease"}
	if err := templateRepo.Create(tmpl); err != nil {
		t.Fatalf("Create(template) error = %v", err)
	}

	if err := favRepo.Create(1, tmpl.ID); err != nil {
		t.Fatalf("Create(favorite) error = %v", err)
	}
	if err := favRepo.Create(1, tmpl.ID); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("Create(favorite duplicate) error = %v, want ErrDuplicateKey", err)
	}

	exists, err := favRepo.Exists(1, tmpl.ID)
	if err != nil || !exists {
		t.Fatalf("Exists() = %v, %v", exists, err)
	}

	list, total, err := favRepo.ListByUser(1, 0, 10)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("ListByUser() = %d, %d, %v", len(list), total, err)
	}

	if err := favRepo.Delete(1, tmpl.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := favRepo.Delete(1, tmpl.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete(missing) error = %v, want ErrNotFound", err)
	}
}
