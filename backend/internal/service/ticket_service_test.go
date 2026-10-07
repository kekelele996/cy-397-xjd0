package service_test

import (
	"testing"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/service"
)

func TestTicketServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		ticketType string
		wantErr bool
	}{
		{name: "labor", ticketType: constants.TicketTypeLabor, wantErr: false},
		{name: "property", ticketType: constants.TicketTypeProperty, wantErr: false},
		{name: "invalid", ticketType: "unknown", wantErr: true},
	}

	repo := newMockTicketRepo()
	svc := service.NewTicketService(repo, testLogger())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := svc.Create(1, dto.CreateTicketRequest{
				Type:        tt.ticketType,
				Title:       "咨询",
				Description: "问题描述",
				Attachments: []string{"evidence.pdf"},
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if ticket.Status != constants.TicketStatusPending {
				t.Fatalf("Create() status = %q, want pending", ticket.Status)
			}
		})
	}
}

func TestTicketServiceReplyTransitions(t *testing.T) {
	repo := newMockTicketRepo()
	svc := service.NewTicketService(repo, testLogger())
	ticket, err := svc.Create(1, dto.CreateTicketRequest{
		Type:        constants.TicketTypeContract,
		Title:       "合同咨询",
		Description: "咨询内容",
	})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	_, err = svc.AddReply(1, ticket.ID, constants.TicketReplyRoleUser, "补充说明", nil)
	if err != nil {
		t.Fatalf("AddReply(user) unexpected error: %v", err)
	}
	got, replies, err := svc.GetForUser(1, ticket.ID)
	if err != nil {
		t.Fatalf("GetForUser() unexpected error: %v", err)
	}
	if got.Status != constants.TicketStatusProcessing {
		t.Fatalf("status after user reply = %q, want processing", got.Status)
	}
	if len(replies) != 1 {
		t.Fatalf("replies = %d, want 1", len(replies))
	}

	_, err = svc.AddReply(1, ticket.ID, constants.TicketReplyRoleLawyer, "法律意见", []string{"answer.pdf"})
	if err != nil {
		t.Fatalf("AddReply(lawyer) unexpected error: %v", err)
	}
	got, _, err = svc.GetForUser(1, ticket.ID)
	if err != nil {
		t.Fatalf("GetForUser() unexpected error: %v", err)
	}
	if got.Status != constants.TicketStatusReplied {
		t.Fatalf("status after lawyer reply = %q, want replied", got.Status)
	}

	if err := svc.Transition(1, ticket.ID, constants.TicketStatusClosed); err != nil {
		t.Fatalf("Transition(closed) unexpected error: %v", err)
	}
	if _, err := svc.AddReply(1, ticket.ID, constants.TicketReplyRoleUser, "再问", nil); err == nil {
		t.Fatal("AddReply() expected error after closed, got nil")
	}
}
