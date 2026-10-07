package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/contractapi/contractapi/internal/model"
)

func TestContractRepository(t *testing.T) {
	repo := NewContractRepository(newTestDB(t))

	contract := &model.Contract{
		UserID:     1,
		TemplateID: 1,
		Title:      "测试合同",
		Status:     "draft",
		Variables:  model.JSONMap{"party_a": "张三"},
	}
	if err := repo.Create(contract); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.FindByIDForUser(contract.ID, 1)
	if err != nil || got.ID != contract.ID {
		t.Fatalf("FindByIDForUser() = %+v, %v", got, err)
	}
	if _, err := repo.FindByIDForUser(contract.ID, 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindByIDForUser(other) error = %v, want ErrNotFound", err)
	}

	list, total, err := repo.ListByUser(1, "draft", 0, 10)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("ListByUser() = %d, %d, %v", len(list), total, err)
	}

	now := time.Now()
	contract.Status = "signed"
	contract.SignedAt = &now
	if err := repo.Update(contract); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	signer := &model.ContractSigner{ContractID: contract.ID, Name: "张三", Role: "甲方", SignedAt: &now}
	if err := repo.AddSigner(signer); err != nil {
		t.Fatalf("AddSigner() error = %v", err)
	}
	signers, err := repo.ListSigners(contract.ID)
	if err != nil || len(signers) != 1 {
		t.Fatalf("ListSigners() = %d, %v", len(signers), err)
	}
}
