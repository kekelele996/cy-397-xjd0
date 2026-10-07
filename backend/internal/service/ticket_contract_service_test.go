package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/service"
)

var errBoom = errors.New("boom")

type failingTicketSyncer struct{ calls int }

func (f *failingTicketSyncer) SyncContractTickets(contractID uint64) (*dto.ContractTicketSyncResult, error) {
	f.calls++
	return nil, errBoom
}

// setupLinkedFixture 准备一份已提交待签署的合同（含 2 个签署方）与工单联动服务。
func setupLinkedFixture(t *testing.T) (*mockContractRepo, *mockTicketRepo, *service.TicketContractService) {
	t.Helper()
	contracts := newMockContractRepo()
	tickets := newMockTicketRepo()
	contract := &model.Contract{
		ID: 1, UserID: 1, TemplateID: 1, Title: "合同甲",
		Status: constants.ContractStatusPendingSign,
	}
	contracts.contracts[1] = contract
	if err := contracts.AddSigner(&model.ContractSigner{ContractID: 1, Name: "张三", Role: "甲方"}); err != nil {
		t.Fatalf("seed signer: %v", err)
	}
	if err := contracts.AddSigner(&model.ContractSigner{ContractID: 1, Name: "李四", Role: "乙方"}); err != nil {
		t.Fatalf("seed signer: %v", err)
	}
	link := service.NewTicketContractService(tickets, contracts, testLogger())
	return contracts, tickets, link
}

func linkedTicket(t *testing.T, tickets *mockTicketRepo, status string) *model.LegalTicket {
	t.Helper()
	ticket := &model.LegalTicket{
		UserID:                  1,
		Type:                    constants.TicketTypeContract,
		Title:                   "纠纷",
		Description:             "内容",
		Status:                  status,
		ContractID:              1,
		ContractStatusSnapshot:  constants.ContractStatusPendingSign,
		ContractSignersSnapshot: model.SignerSnapshots{{Name: "张三", Role: "甲方"}, {Name: "李四", Role: "乙方"}},
	}
	if err := tickets.Create(ticket); err != nil {
		t.Fatalf("seed ticket: %v", err)
	}
	return ticket
}

func TestCreateTicketRegistersContractSnapshot(t *testing.T) {
	contracts, tickets, _ := setupLinkedFixture(t)
	svc := service.NewTicketService(tickets, contracts, testLogger())
	ticket, err := svc.Create(1, dto.CreateTicketRequest{
		Type: constants.TicketTypeContract, Title: "纠纷",
		Description: "关于合同", ContractID: 1,
	})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if ticket.ContractID != 1 || ticket.ContractStatusSnapshot != constants.ContractStatusPendingSign {
		t.Fatalf("snapshot not registered: %+v", ticket)
	}
	if len(ticket.ContractSignersSnapshot) != 2 {
		t.Fatalf("signers snapshot = %d, want 2", len(ticket.ContractSignersSnapshot))
	}

	// 关联不存在/不属于自己的合同应报错。
	if _, err := svc.Create(1, dto.CreateTicketRequest{
		Type: constants.TicketTypeContract, Title: "x", Description: "y", ContractID: 99,
	}); err == nil {
		t.Fatal("Create() with missing contract expected error, got nil")
	}
}

func TestContractSettleMarksOpenTicketsPendingReview(t *testing.T) {
	_, tickets, link := setupLinkedFixture(t)
	open1 := linkedTicket(t, tickets, constants.TicketStatusProcessing)
	open2 := linkedTicket(t, tickets, constants.TicketStatusReplied)
	closed := linkedTicket(t, tickets, constants.TicketStatusClosed)

	res, err := link.SyncContractTickets(1)
	if err != nil {
		t.Fatalf("SyncContractTickets() unexpected error: %v", err)
	}
	if res.Affected != 2 || len(res.PendingReviewIDs) != 2 {
		t.Fatalf("sync result = %+v, want 2 open", res)
	}
	if tickets.tickets[open1.ID].Status != constants.TicketStatusPendingReview ||
		tickets.tickets[open2.ID].Status != constants.TicketStatusPendingReview {
		t.Fatal("open tickets should be pending_review")
	}
	if tickets.tickets[closed.ID].Status != constants.TicketStatusClosed {
		t.Fatal("closed ticket must stay closed")
	}

	// 重试幂等：不会重复计数，也不会把挂起工单改掉。
	res2, err := link.SyncContractTickets(1)
	if err != nil || res2.Affected != 0 {
		t.Fatalf("retry sync = %+v, %v; want affected 0", res2, err)
	}
}

func TestSignFailureDoesNotRollBackContract(t *testing.T) {
	contracts, _, _ := setupLinkedFixture(t)
	syncer := &failingTicketSyncer{}
	svc := service.NewContractService(contracts, newMockTemplateRepo(), service.NewPDFService(testLogger()), syncer, testLogger())
	if err := svc.Sign(1, 1, "张三", "甲方", ""); err != nil {
		t.Fatalf("Sign() should succeed even when ticket sync fails: %v", err)
	}
	if syncer.calls != 1 {
		t.Fatalf("syncer calls = %d, want 1", syncer.calls)
	}
	if contracts.contracts[1].Status != constants.ContractStatusSigned {
		t.Fatalf("contract status = %q, want signed despite sync failure", contracts.contracts[1].Status)
	}
}

func TestCloseWithReconcile(t *testing.T) {
	t.Run("matched closes", func(t *testing.T) {
		_, tickets, link := setupLinkedFixture(t)
		tk := linkedTicket(t, tickets, constants.TicketStatusReplied)
		res, err := link.CloseWithReconcile(1, tk.ID)
		if err != nil || !res.Closed {
			t.Fatalf("CloseWithReconcile() = %+v, %v", res, err)
		}
	})

	t.Run("status mismatch holds", func(t *testing.T) {
		contracts, tickets, link := setupLinkedFixture(t)
		tk := linkedTicket(t, tickets, constants.TicketStatusReplied)
		contracts.contracts[1].Status = constants.ContractStatusSigned

		_, err := link.CloseWithReconcile(1, tk.ID)
		appErr, ok := dto.IsAppError(err)
		if !ok {
			t.Fatalf("expected AppError, got %v", err)
		}
		if tickets.tickets[tk.ID].Status != constants.TicketStatusOnHold {
			t.Fatalf("ticket status = %q, want on_hold", tickets.tickets[tk.ID].Status)
		}
		payload, ok := appErr.Data.(map[string]any)
		if !ok {
			t.Fatalf("error data = %#v", appErr.Data)
		}
		list, _ := payload["mismatches"].([]dto.TicketMismatch)
		if len(list) != 1 || list[0].Differences[0].Field != "contract_status" {
			t.Fatalf("mismatch payload = %#v", list)
		}
	})

	t.Run("signers mismatch holds", func(t *testing.T) {
		contracts, tickets, link := setupLinkedFixture(t)
		tk := linkedTicket(t, tickets, constants.TicketStatusProcessing)
		if err := contracts.AddSigner(&model.ContractSigner{ContractID: 1, Name: "王五", Role: "见证方"}); err != nil {
			t.Fatalf("add signer: %v", err)
		}
		if _, err := link.CloseWithReconcile(1, tk.ID); err == nil {
			t.Fatal("expected mismatch error after new signer added")
		}
		if tickets.tickets[tk.ID].Status != constants.TicketStatusOnHold {
			t.Fatal("ticket should be on_hold")
		}
	})

	t.Run("missing contract holds", func(t *testing.T) {
		contracts, tickets, link := setupLinkedFixture(t)
		tk := linkedTicket(t, tickets, constants.TicketStatusProcessing)
		delete(contracts.contracts, 1)
		if _, err := link.CloseWithReconcile(1, tk.ID); err == nil {
			t.Fatal("expected mismatch error when contract deleted")
		}
		if tickets.tickets[tk.ID].Status != constants.TicketStatusOnHold {
			t.Fatal("ticket should be on_hold")
		}
	})
}

func TestReviewRefreshesSnapshotAndUnblocksClose(t *testing.T) {
	contracts, tickets, link := setupLinkedFixture(t)
	tk := linkedTicket(t, tickets, constants.TicketStatusOnHold)
	now := time.Now()
	contracts.contracts[1].Status = constants.ContractStatusSigned
	contracts.contracts[1].SignedAt = &now

	res, err := link.Review(1, tk.ID, "已电话核实")
	if err != nil {
		t.Fatalf("Review() unexpected error: %v", err)
	}
	if res.Status != constants.TicketStatusProcessing || res.CurrentStatus != constants.ContractStatusSigned {
		t.Fatalf("review result = %+v", res)
	}
	stored := tickets.tickets[tk.ID]
	if stored.ContractStatusSnapshot != constants.ContractStatusSigned {
		t.Fatalf("snapshot after review = %q", stored.ContractStatusSnapshot)
	}

	// 快照刷新后再次关单应成功。
	if _, err := link.CloseWithReconcile(1, tk.ID); err != nil {
		t.Fatalf("CloseWithReconcile() after review unexpected error: %v", err)
	}
	if tickets.tickets[tk.ID].Status != constants.TicketStatusClosed {
		t.Fatal("ticket should close after review")
	}
}

func TestReviewRejectsWrongStatus(t *testing.T) {
	_, tickets, link := setupLinkedFixture(t)
	tk := linkedTicket(t, tickets, constants.TicketStatusProcessing)
	if _, err := link.Review(1, tk.ID, ""); err == nil {
		t.Fatal("Review() expected error for non-reviewable ticket")
	}
}

func TestBatchCloseMixed(t *testing.T) {
	contracts, tickets, link := setupLinkedFixture(t)
	matched := linkedTicket(t, tickets, constants.TicketStatusReplied)
	mismatched := linkedTicket(t, tickets, constants.TicketStatusReplied)
	contracts.contracts[1].Status = constants.ContractStatusExpired

	res, err := link.BatchClose(1, []uint64{matched.ID, mismatched.ID})
	if err != nil {
		t.Fatalf("BatchClose() unexpected error: %v", err)
	}
	if len(res.Closed) != 0 {
		// 两张工单快照相同，合同状态都变了，理论上都对不上
		t.Fatalf("closed = %v, want none", res.Closed)
	}
	if len(res.Mismatch) != 2 {
		t.Fatalf("mismatch = %d, want 2", len(res.Mismatch))
	}
	for id := range tickets.tickets {
		if tickets.tickets[id].Status != constants.TicketStatusOnHold {
			t.Fatalf("ticket %d status = %q, want on_hold", id, tickets.tickets[id].Status)
		}
	}
}

func TestBatchClosePartialMatch(t *testing.T) {
	contracts, tickets, link := setupLinkedFixture(t)
	// 合同 2 状态未变；合同 1 已变更
	contract2 := &model.Contract{ID: 2, UserID: 1, Status: constants.ContractStatusSigned}
	contracts.contracts[2] = contract2
	okTicket := &model.LegalTicket{
		UserID: 1, Type: constants.TicketTypeContract, Title: "ok", Description: "d",
		Status: constants.TicketStatusReplied, ContractID: 2,
		ContractStatusSnapshot: constants.ContractStatusSigned,
	}
	if err := tickets.Create(okTicket); err != nil {
		t.Fatalf("seed: %v", err)
	}
	badTicket := linkedTicket(t, tickets, constants.TicketStatusReplied)
	contracts.contracts[1].Status = constants.ContractStatusExpired

	res, err := link.BatchClose(1, []uint64{okTicket.ID, badTicket.ID})
	if err != nil {
		t.Fatalf("BatchClose() unexpected error: %v", err)
	}
	if len(res.Closed) != 1 || res.Closed[0] != okTicket.ID {
		t.Fatalf("closed = %v, want [%d]", res.Closed, okTicket.ID)
	}
	if len(res.Mismatch) != 1 || res.Mismatch[0].TicketID != badTicket.ID {
		t.Fatalf("mismatch = %+v", res.Mismatch)
	}
	if tickets.tickets[badTicket.ID].Status != constants.TicketStatusOnHold {
		t.Fatal("mismatched ticket should be on_hold")
	}
}

func TestBuildSnapshotView(t *testing.T) {
	contracts, tickets, link := setupLinkedFixture(t)
	tk := linkedTicket(t, tickets, constants.TicketStatusProcessing)

	view, err := link.BuildSnapshotView(tk)
	if err != nil {
		t.Fatalf("BuildSnapshotView() error = %v", err)
	}
	if !view.Reconciled || view.CurrentStatus != constants.ContractStatusPendingSign ||
		len(view.CurrentSigners) != 2 {
		t.Fatalf("view = %+v", view)
	}

	contracts.contracts[1].Status = constants.ContractStatusSigned
	view, err = link.BuildSnapshotView(tk)
	if err != nil || view.Reconciled {
		t.Fatalf("after contract change view = %+v, err %v; want reconciled=false", view, err)
	}

	// 未关联合同的旧工单。
	legacy := &model.LegalTicket{ID: 9, UserID: 1, Type: "other", Title: "t", Description: "d"}
	legacyView, err := link.BuildSnapshotView(legacy)
	if err != nil || !legacyView.Reconciled || legacyView.ContractID != 0 {
		t.Fatalf("legacy view = %+v, err %v", legacyView, err)
	}
}
