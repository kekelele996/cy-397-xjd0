package repository

import (
	"testing"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/model"
)

func TestTicketContractLinkRepository(t *testing.T) {
	db := newTestDB(t)
	contractRepo := NewContractRepository(db)
	ticketRepo := NewTicketRepository(db)

	contract := &model.Contract{
		ContractNo: "HT-2026-000010", UserID: 1, TemplateID: 1, Title: "t",
		Status: constants.ContractStatusSigned, Variables: model.JSONMap{},
	}
	if err := contractRepo.Create(contract); err != nil {
		t.Fatalf("create contract: %v", err)
	}
	gotContract, err := contractRepo.FindByContractNo("HT-2026-000010")
	if err != nil || gotContract.ID != contract.ID {
		t.Fatalf("FindByContractNo() = %+v, %v", gotContract, err)
	}

	open := &model.LegalTicket{UserID: 1, Type: "contract", Title: "open", Description: "d",
		Status: "processing", ContractNo: "HT-2026-000010", ContractStatusSnapshot: "pending_signed"}
	closed := &model.LegalTicket{UserID: 1, Type: "contract", Title: "closed", Description: "d",
		Status: "closed", ContractNo: "HT-2026-000010", ContractStatusSnapshot: "pending_signed"}
	legacy := &model.LegalTicket{UserID: 1, Type: "other", Title: "legacy", Description: "d", Status: "pending"}
	for _, tk := range []*model.LegalTicket{open, closed, legacy} {
		if err := ticketRepo.Create(tk); err != nil {
			t.Fatalf("create ticket: %v", err)
		}
	}

	marked, err := ticketRepo.MarkReviewPendingByContractNo("HT-2026-000010")
	if err != nil || marked != 1 {
		t.Fatalf("MarkReviewPendingByContractNo() = %d, %v", marked, err)
	}
	updated, err := ticketRepo.FindByID(open.ID)
	if err != nil || updated.Status != constants.TicketStatusReviewPending {
		t.Fatalf("open ticket status = %q, %v", updated.Status, err)
	}
	stillClosed, _ := ticketRepo.FindByID(closed.ID)
	if stillClosed.Status != "closed" {
		t.Fatalf("closed ticket status = %q, want closed", stillClosed.Status)
	}

	candidates, err := ticketRepo.ListBackfillCandidates(100)
	if err != nil || len(candidates) != 1 || candidates[0].ID != legacy.ID {
		t.Fatalf("ListBackfillCandidates() = %+v, %v", candidates, err)
	}
	if err := ticketRepo.AttachContract(legacy.ID, "HT-2026-000010", "signed"); err != nil {
		t.Fatalf("AttachContract() = %v", err)
	}
	attached, _ := ticketRepo.FindByID(legacy.ID)
	if attached.ContractNo != "HT-2026-000010" || attached.ContractStatusSnapshot != "signed" {
		t.Fatalf("attached ticket = %+v", attached)
	}

	if err := ticketRepo.RecordBackfail(999, "no_contract_no_in_description", ""); err != nil {
		t.Fatalf("RecordBackfail() = %v", err)
	}
	// 重复登记应更新而非报错。
	if err := ticketRepo.RecordBackfail(999, "contract_not_found", "HT-2026-000099"); err != nil {
		t.Fatalf("RecordBackfail() second call = %v", err)
	}
	backfails, total, err := ticketRepo.ListBackfails(nil, 0, 10)
	if err != nil || total != 1 || len(backfails) != 1 {
		t.Fatalf("ListBackfails() = %+v, %d, %v", backfails, total, err)
	}
	if backfails[0].Reason != "contract_not_found" || backfails[0].ExtractedNo != "HT-2026-000099" {
		t.Fatalf("backfail = %+v", backfails[0])
	}

	byIDs, err := ticketRepo.ListByIDs([]uint64{open.ID, closed.ID})
	if err != nil || len(byIDs) != 2 {
		t.Fatalf("ListByIDs() = %+v, %v", byIDs, err)
	}
}
