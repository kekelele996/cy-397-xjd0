package repository

import (
	"testing"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/model"
)

func seedTicket(t *testing.T, repo TicketRepository, ticket *model.LegalTicket) *model.LegalTicket {
	t.Helper()
	if err := repo.Create(ticket); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return ticket
}

func TestTicketRepositoryContractLink(t *testing.T) {
	repo := NewTicketRepository(newTestDB(t))

	seedTicket(t, repo, &model.LegalTicket{
		UserID:                  1,
		Type:                    "contract",
		Title:                   "t1",
		Description:             "d",
		Status:                  constants.TicketStatusProcessing,
		ContractID:              10,
		ContractStatusSnapshot:  constants.ContractStatusPendingSign,
		ContractSignersSnapshot: model.SignerSnapshots{{Name: "张三", Role: "甲方"}},
	})
	seedTicket(t, repo, &model.LegalTicket{
		UserID:                  1,
		Type:                    "contract",
		Title:                   "t2",
		Description:             "d",
		Status:                  constants.TicketStatusReplied,
		ContractID:              10,
		ContractStatusSnapshot:  constants.ContractStatusPendingSign,
		ContractSignersSnapshot: model.SignerSnapshots{{Name: "张三", Role: "甲方"}},
	})
	closed := seedTicket(t, repo, &model.LegalTicket{
		UserID:      1,
		Type:        "contract",
		Title:       "t3",
		Description: "d",
		Status:      constants.TicketStatusClosed,
		ContractID:  10,
	})
	seedTicket(t, repo, &model.LegalTicket{
		UserID:      2,
		Type:        "other",
		Title:       "legacy",
		Description: "d",
		Status:      constants.TicketStatusPending,
		ContractID:  0,
	})

	open, err := repo.ListOpenByContract(10)
	if err != nil || len(open) != 2 {
		t.Fatalf("ListOpenByContract() = %d, %v; want 2", len(open), err)
	}

	affected, err := repo.MarkOpenTicketsPendingReview(10)
	if err != nil || affected != 2 {
		t.Fatalf("MarkOpenTicketsPendingReview() = %d, %v; want 2", affected, err)
	}
	open2, _ := repo.ListOpenByContract(10)
	for _, tk := range open2 {
		if tk.Status != constants.TicketStatusPendingReview {
			t.Fatalf("ticket %d status = %q, want pending_review", tk.ID, tk.Status)
		}
	}
	closedGot, err := repo.FindByID(closed.ID)
	if err != nil || closedGot.Status != constants.TicketStatusClosed {
		t.Fatalf("closed ticket changed: %+v, %v", closedGot, err)
	}

	// 重试幂等：待复核不会被重复更新。
	affected2, err := repo.MarkOpenTicketsPendingReview(10)
	if err != nil || affected2 != 0 {
		t.Fatalf("second MarkOpenTicketsPendingReview() = %d, %v; want 0", affected2, err)
	}

	// JSON 快照往返。
	got, err := repo.FindByID(open[0].ID)
	if err != nil || len(got.ContractSignersSnapshot) != 1 ||
		got.ContractSignersSnapshot[0].Name != "张三" {
		t.Fatalf("snapshot round trip = %+v, %v", got, err)
	}
}

func TestTicketRepositoryListWithoutContract(t *testing.T) {
	repo := NewTicketRepository(newTestDB(t))
	seedTicket(t, repo, &model.LegalTicket{
		UserID: 1, Type: "other", Title: "legacy1", Description: "d",
		Status: constants.TicketStatusPending, ContractID: 0,
	})
	seedTicket(t, repo, &model.LegalTicket{
		UserID: 1, Type: "contract", Title: "linked", Description: "d",
		Status: constants.TicketStatusPending, ContractID: 5,
	})
	seedTicket(t, repo, &model.LegalTicket{
		UserID: 2, Type: "other", Title: "legacy2", Description: "d",
		Status: constants.TicketStatusPending, ContractID: 0,
	})

	list, err := repo.ListWithoutContract(0, 100)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListWithoutContract() = %d, %v; want 2", len(list), err)
	}
	for _, tk := range list {
		if tk.ContractID != 0 {
			t.Fatalf("ticket %d should have no contract", tk.ID)
		}
	}

	byIDs, err := repo.ListByIDs([]uint64{list[0].ID, 999})
	if err != nil || len(byIDs) != 1 {
		t.Fatalf("ListByIDs() = %d, %v; want 1", len(byIDs), err)
	}
	if empty, err := repo.ListByIDs(nil); err != nil || len(empty) != 0 {
		t.Fatalf("ListByIDs(nil) = %d, %v; want empty", len(empty), err)
	}
}
