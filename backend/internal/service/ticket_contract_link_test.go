package service_test

import (
	"testing"
	"time"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/service"
)

func newSettledContractFixture() (*mockContractRepo, *mockTicketRepo, *service.ContractService) {
	_, contracts, contractSvc := newContractFixture()
	contract, err := contractSvc.Create(1, dto.CreateContractRequest{
		TemplateID: 1,
		Title:      "纠纷关联合同",
		Variables:  map[string]string{"party_a": "甲", "party_b": "乙", "amount": "1000"},
	})
	if err != nil {
		panic(err)
	}
	// 固定创建年份，使合同编号形如 HT-2026-000001。
	contract.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	contract.ContractNo = ""
	if err := contracts.Update(contract); err != nil {
		panic(err)
	}
	contract.ContractNo = "HT-2026-000001"
	if err := contracts.Update(contract); err != nil {
		panic(err)
	}
	if err := contractSvc.Submit(1, contract.ID, []dto.SignerInput{{Name: "甲", Role: "甲方"}}); err != nil {
		panic(err)
	}
	ticketRepo := newMockTicketRepo()
	// 用同一个 contracts 仓储重建带 ticketRepo 的服务，复用已落定数据。
	svc := service.NewContractService(contracts, ticketRepo, newMockTemplateRepo(),
		service.NewPDFService(testLogger()), testLogger())
	return contracts, ticketRepo, svc
}

func createTicketWithContract(t *testing.T, ticketSvc *service.TicketService, contractNo, snapshot string) *model.LegalTicket {
	t.Helper()
	ticket, err := ticketSvc.Create(1, dto.CreateTicketRequest{
		Type:        constants.TicketTypeContract,
		Title:       "合同纠纷",
		Description: "对方违约",
		ContractNo:  contractNo,
	})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if ticket.ContractStatusSnapshot != snapshot {
		t.Fatalf("snapshot = %q, want %q", ticket.ContractStatusSnapshot, snapshot)
	}
	return ticket
}

func TestTicketCreateRegistersContractSnapshot(t *testing.T) {
	contracts := newMockContractRepo()
	contracts.Create(&model.Contract{ContractNo: "HT-2026-000001", UserID: 9, Status: constants.ContractStatusPendingSign})
	svc := service.NewTicketService(newMockTicketRepo(), contracts, testLogger())

	ticket := createTicketWithContract(t, svc, "HT-2026-000001", constants.ContractStatusPendingSign)
	if ticket.ContractNo != "HT-2026-000001" {
		t.Fatalf("contract_no = %q", ticket.ContractNo)
	}

	if _, err := svc.Create(1, dto.CreateTicketRequest{
		Type: constants.TicketTypeOther, Title: "t", Description: "d", ContractNo: "BAD-1",
	}); err == nil {
		t.Fatal("expected validation error for malformed contract_no")
	}
	if _, err := svc.Create(1, dto.CreateTicketRequest{
		Type: constants.TicketTypeOther, Title: "t", Description: "d", ContractNo: "HT-2026-999999",
	}); err == nil {
		t.Fatal("expected not found error for missing contract")
	}
}

func TestContractSignMarksLinkedTicketsReviewPending(t *testing.T) {
	_, ticketRepo, contractSvc := newSettledContractFixture()
	contractNo := "HT-2026-000001"
	// 直接在 ticket repo 中构造两笔关联工单（未关），外加已关工单和无关工单。
	open1 := &model.LegalTicket{UserID: 1, Type: "contract", Title: "t1", Description: "d",
		Status: "processing", ContractNo: contractNo, ContractStatusSnapshot: "pending_signed"}
	open2 := &model.LegalTicket{UserID: 1, Type: "contract", Title: "t2", Description: "d",
		Status: "replied", ContractNo: contractNo, ContractStatusSnapshot: "pending_signed"}
	closed := &model.LegalTicket{UserID: 1, Type: "contract", Title: "t3", Description: "d",
		Status: "closed", ContractNo: contractNo, ContractStatusSnapshot: "pending_signed"}
	other := &model.LegalTicket{UserID: 1, Type: "contract", Title: "t4", Description: "d", Status: "processing"}
	for _, tk := range []*model.LegalTicket{open1, open2, closed, other} {
		if err := ticketRepo.Create(tk); err != nil {
			t.Fatalf("seed ticket: %v", err)
		}
	}

	result, err := contractSvc.Sign(1, 1, "甲", "甲方", "")
	if err != nil {
		t.Fatalf("Sign() unexpected error: %v", err)
	}
	if result.ReviewMarked != 2 || result.ReviewSyncError {
		t.Fatalf("settle result = %+v, want 2 marked, no sync error", result)
	}
	for id, want := range map[uint64]string{open1.ID: "review_pending", open2.ID: "review_pending",
		closed.ID: "closed", other.ID: "processing"} {
		got, err := ticketRepo.FindByID(id)
		if err != nil || got.Status != want {
			t.Fatalf("ticket %d status = %q (%v), want %q", id, got.Status, err, want)
		}
	}
}

func TestCloseTicketReconcileMatchesAndMismatches(t *testing.T) {
	contracts := newMockContractRepo()
	if err := contracts.Create(&model.Contract{ContractNo: "HT-2026-000001", UserID: 1,
		Status: constants.ContractStatusPendingSign}); err != nil {
		t.Fatal(err)
	}
	ticketRepo := newMockTicketRepo()
	svc := service.NewTicketService(ticketRepo, contracts, testLogger())

	// 快照与现状一致 -> 允许关单。
	okTicket := createTicketWithContract(t, svc, "HT-2026-000001", constants.ContractStatusPendingSign)
	if err := svc.Transition(1, okTicket.ID, constants.TicketStatusClosed); err != nil {
		t.Fatalf("Transition(closed) matched case unexpected error: %v", err)
	}

	// 合同状态变化 -> 关单对账失败，工单挂起。
	badTicket := createTicketWithContract(t, svc, "HT-2026-000001", constants.ContractStatusPendingSign)
	target, _ := contracts.FindByID(1)
	target.Status = constants.ContractStatusSigned
	if err := contracts.Update(target); err != nil {
		t.Fatal(err)
	}
	err := svc.Transition(1, badTicket.ID, constants.TicketStatusClosed)
	if err == nil {
		t.Fatal("expected reconcile error, got nil")
	}
	appErr, isApp := dto.IsAppError(err)
	if !isApp || appErr.Code != constants.CodeReconcileFailed {
		t.Fatalf("error = %v, want reconcile failed", err)
	}
	mismatches, ok := appErr.Details.([]dto.ReconcileMismatch)
	if !ok || len(mismatches) != 1 {
		t.Fatalf("details = %#v", appErr.Details)
	}
	m := mismatches[0]
	if m.Reason != "status_mismatch" || m.RegisteredStatus != constants.ContractStatusPendingSign ||
		m.CurrentStatus != constants.ContractStatusSigned {
		t.Fatalf("mismatch = %+v", m)
	}
	got, _ := ticketRepo.FindByID(badTicket.ID)
	if got.Status != constants.TicketStatusReviewPending {
		t.Fatalf("ticket status after blocked close = %q, want review_pending", got.Status)
	}

	// 待复核状态下恢复处理后才能继续流转。
	if err := svc.Transition(1, badTicket.ID, constants.TicketStatusClosed); err == nil {
		t.Fatal("review_pending -> closed without resume still runs reconcile and should fail")
	}
	// 复核通过：按合同库现状刷新快照并恢复处理，之后关单对账即一致。
	if err := svc.ResumeAfterReview(1, badTicket.ID); err != nil {
		t.Fatalf("ResumeAfterReview() unexpected error: %v", err)
	}
	refreshed, _ := ticketRepo.FindByID(badTicket.ID)
	if refreshed.Status != constants.TicketStatusProcessing ||
		refreshed.ContractStatusSnapshot != constants.ContractStatusSigned {
		t.Fatalf("refreshed ticket = %+v", refreshed)
	}
	if err := svc.Transition(1, badTicket.ID, constants.TicketStatusClosed); err != nil {
		t.Fatalf("close after resume: %v", err)
	}

	// 合同被删除 -> 对不上，挂起。
	other := createTicketWithContract(t, svc, "HT-2026-000001", constants.ContractStatusSigned)
	delete(contracts.contracts, uint64(1))
	err = svc.Transition(1, other.ID, constants.TicketStatusClosed)
	if err == nil {
		t.Fatal("expected reconcile error when contract missing")
	}
	appErr, isApp = dto.IsAppError(err)
	if !isApp || appErr.Code != constants.CodeReconcileFailed {
		t.Fatalf("error = %v, want reconcile failed", err)
	}
	mismatches, ok = appErr.Details.([]dto.ReconcileMismatch)
	if !ok || len(mismatches) != 1 || mismatches[0].Reason != "contract_not_found" {
		t.Fatalf("details = %#v", appErr.Details)
	}
	suspended, _ := ticketRepo.FindByID(other.ID)
	if suspended.Status != constants.TicketStatusReviewPending {
		t.Fatalf("ticket status = %q, want review_pending", suspended.Status)
	}
}

func TestBatchCloseReconcile(t *testing.T) {
	contracts := newMockContractRepo()
	if err := contracts.Create(&model.Contract{ContractNo: "HT-2026-000007", UserID: 1,
		Status: constants.ContractStatusSigned}); err != nil {
		t.Fatal(err)
	}
	if err := contracts.Create(&model.Contract{ContractNo: "HT-2026-000008", UserID: 1,
		Status: constants.ContractStatusSigned}); err != nil {
		t.Fatal(err)
	}
	ticketRepo := newMockTicketRepo()
	svc := service.NewTicketService(ticketRepo, contracts, testLogger())

	// 合同 007 一直是已签署 -> 对得上，可关。
	matched := createTicketWithContract(t, svc, "HT-2026-000007", constants.ContractStatusSigned)
	// 合同 008 登记后过期 -> 对不上，挂起。
	mismatch := createTicketWithContract(t, svc, "HT-2026-000008", constants.ContractStatusSigned)
	target, _ := contracts.FindByContractNo("HT-2026-000008")
	target.Status = constants.ContractStatusExpired
	if err := contracts.Update(target); err != nil {
		t.Fatal(err)
	}

	result, err := svc.BatchClose(1, []uint64{matched.ID, mismatch.ID})
	if err != nil {
		t.Fatalf("BatchClose() unexpected error: %v", err)
	}
	if len(result.Closed) != 1 || result.Closed[0] != matched.ID {
		t.Fatalf("closed = %v", result.Closed)
	}
	if len(result.Suspended) != 1 || result.Suspended[0].TicketID != mismatch.ID ||
		result.Suspended[0].Reason != "status_mismatch" {
		t.Fatalf("suspended = %+v", result.Suspended)
	}
	got, _ := ticketRepo.FindByID(mismatch.ID)
	if got.Status != constants.TicketStatusReviewPending {
		t.Fatalf("mismatched ticket = %q, want review_pending", got.Status)
	}
}

func TestBackfillContractNo(t *testing.T) {
	contracts := newMockContractRepo()
	if err := contracts.Create(&model.Contract{ContractNo: "HT-2026-000042", UserID: 2,
		Status: constants.ContractStatusSigned}); err != nil {
		t.Fatal(err)
	}
	ticketRepo := newMockTicketRepo()
	svc := service.NewTicketService(ticketRepo, contracts, testLogger())

	// 描述中带编号且合同存在 -> 回填。
	resolvable := &model.LegalTicket{UserID: 1, Type: "contract", Title: "旧工单",
		Description: "合同编号 HT-2026-000042 发生纠纷", Status: "pending"}
	// 描述中没有编号 -> 回填失败清单。
	noNumber := &model.LegalTicket{UserID: 1, Type: "other", Title: "旧工单2",
		Description: "口头约定没有编号", Status: "pending"}
	// 编号在合同库不存在 -> 回填失败清单。
	missingContract := &model.LegalTicket{UserID: 1, Type: "contract", Title: "旧工单3",
		Description: "HT-2026-000043 找不到", Status: "pending"}
	for _, tk := range []*model.LegalTicket{resolvable, noNumber, missingContract} {
		if err := ticketRepo.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	result, err := svc.BackfillContractNo()
	if err != nil {
		t.Fatalf("BackfillContractNo() unexpected error: %v", err)
	}
	if result.Total != 3 || result.Backfilled != 1 || result.Failed != 2 {
		t.Fatalf("backfill result = %+v", result)
	}
	updated, _ := ticketRepo.FindByID(resolvable.ID)
	if updated.ContractNo != "HT-2026-000042" || updated.ContractStatusSnapshot != constants.ContractStatusSigned {
		t.Fatalf("backfilled ticket = %+v", updated)
	}
	backfails, total, err := ticketRepo.ListBackfails(nil, 0, 100)
	if err != nil || total != 2 || len(backfails) != 2 {
		t.Fatalf("backfails = %d, total = %d, err = %v", len(backfails), total, err)
	}
}

func TestTicketDetailCarriesCurrentContract(t *testing.T) {
	contracts := newMockContractRepo()
	contract := &model.Contract{ContractNo: "HT-2026-000005", UserID: 2,
		Status: constants.ContractStatusPendingSign, Title: "租约"}
	if err := contracts.Create(contract); err != nil {
		t.Fatal(err)
	}
	svc := service.NewTicketService(newMockTicketRepo(), contracts, testLogger())
	ticket := createTicketWithContract(t, svc, "HT-2026-000005", constants.ContractStatusPendingSign)

	detail, err := svc.GetDetailForUser(1, ticket.ID)
	if err != nil {
		t.Fatalf("GetDetailForUser() unexpected error: %v", err)
	}
	if detail.Contract == nil || !detail.Contract.Found ||
		detail.Contract.Status != constants.ContractStatusPendingSign {
		t.Fatalf("contract snapshot = %+v", detail.Contract)
	}

	// 合同库状态变化后，详情实时反映新状态。
	contract.Status = constants.ContractStatusSigned
	if err := contracts.Update(contract); err != nil {
		t.Fatal(err)
	}
	detail, _ = svc.GetDetailForUser(1, ticket.ID)
	if detail.Contract.Status != constants.ContractStatusSigned {
		t.Fatalf("current status = %q, want signed", detail.Contract.Status)
	}
	if detail.RegisteredContractStatus != constants.ContractStatusPendingSign {
		t.Fatalf("registered snapshot = %q, should remain pending_signed", detail.RegisteredContractStatus)
	}
}

func TestRecheckRetriesOnlyTicketBatch(t *testing.T) {
	contracts, ticketRepo, contractSvc := newSettledContractFixture()
	// 合同已经是 pending_signed；先挂一个工单。
	ticket := &model.LegalTicket{UserID: 1, Type: "contract", Title: "t", Description: "d",
		Status: "processing", ContractNo: "HT-2026-000001", ContractStatusSnapshot: "draft"}
	if err := ticketRepo.Create(ticket); err != nil {
		t.Fatal(err)
	}
	result, err := contractSvc.RecheckContractTickets(1, 1)
	if err != nil {
		t.Fatalf("RecheckContractTickets() unexpected error: %v", err)
	}
	if result.ReviewMarked != 1 {
		t.Fatalf("recheck result = %+v", result)
	}
	got, err := ticketRepo.FindByID(ticket.ID)
	if err != nil || got.Status != constants.TicketStatusReviewPending {
		t.Fatalf("ticket after recheck = %q, %v", got.Status, err)
	}
	// 合同库状态未被补偿操作改动。
	contract, err := contracts.FindByID(1)
	if err != nil || contract.Status != constants.ContractStatusPendingSign {
		t.Fatalf("contract status = %q, %v; contract store must stay untouched", contract.Status, err)
	}
}
