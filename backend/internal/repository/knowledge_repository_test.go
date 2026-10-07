package repository

import (
	"errors"
	"testing"

	"github.com/contractapi/contractapi/internal/model"
)

func TestKnowledgeRepository(t *testing.T) {
	repo := NewKnowledgeRepository(newTestDB(t))

	faq := &model.KnowledgeFAQ{Category: "劳动纠纷", Question: "试用期多久？", Answer: "最长六个月。"}
	if err := repo.Create(faq); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	list, total, err := repo.List("劳动纠纷", "试用期", 0, 10)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List() = %d, %d, %v", len(list), total, err)
	}

	got, err := repo.FindByID(faq.ID)
	if err != nil || got.Question != faq.Question {
		t.Fatalf("FindByID() = %+v, %v", got, err)
	}

	faq.Answer = "更新后答案"
	if err := repo.Update(faq); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := repo.Delete(faq.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(faq.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindByID(deleted) error = %v, want ErrNotFound", err)
	}
}
