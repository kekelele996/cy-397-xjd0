package repository

import (
	"testing"

	"github.com/contractapi/contractapi/internal/model"
)

func TestTicketRepository(t *testing.T) {
	repo := NewTicketRepository(newTestDB(t))

	ticket := &model.LegalTicket{
		UserID:      1,
		Type:        "contract",
		Title:       "咨询",
		Description: "问题",
		Status:      "pending",
		Attachments: model.StringSlice{"a.pdf"},
	}
	if err := repo.Create(ticket); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	list, total, err := repo.ListByUser(1, "", 0, 10)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("ListByUser() = %d, %d, %v", len(list), total, err)
	}

	reply := &model.TicketReply{TicketID: ticket.ID, Role: "lawyer", Content: "答复", Attachments: model.StringSlice{"b.pdf"}}
	if err := repo.AddReply(reply); err != nil {
		t.Fatalf("AddReply() error = %v", err)
	}
	replies, err := repo.ListReplies(ticket.ID)
	if err != nil || len(replies) != 1 {
		t.Fatalf("ListReplies() = %d, %v", len(replies), err)
	}

	ticket.Status = "replied"
	if err := repo.Update(ticket); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	got, err := repo.FindByID(ticket.ID)
	if err != nil || got.Status != "replied" {
		t.Fatalf("FindByID() = %+v, %v", got, err)
	}
}
