package repository

import (
	"errors"
	"testing"

	"github.com/contractapi/contractapi/internal/model"
)

func TestTemplateRepository(t *testing.T) {
	repo := NewTemplateRepository(newTestDB(t))

	tmpl := &model.ContractTemplate{Code: "lease", Name: "房屋租赁合同", Category: "lease"}
	if err := repo.Create(tmpl); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repo.Create(&model.ContractTemplate{Code: "lease", Name: "dup"}); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("Create(duplicate) error = %v, want ErrDuplicateKey", err)
	}

	got, err := repo.FindByID(tmpl.ID)
	if err != nil || got.Code != "lease" {
		t.Fatalf("FindByID() = %+v, %v", got, err)
	}

	list, total, err := repo.List("lease", "租赁", 0, 10)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List() = %d, %d, %v", len(list), total, err)
	}

	tmpl.Name = "租赁合同"
	if err := repo.Update(tmpl); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := repo.Delete(tmpl.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(tmpl.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindByID(deleted) error = %v, want ErrNotFound", err)
	}
}
