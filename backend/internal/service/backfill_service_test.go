package service_test

import (
	"testing"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/service"
)

func TestBackfillServiceRun(t *testing.T) {
	contracts := newMockContractRepo()
	contracts.contracts[7] = &model.Contract{ID: 7, UserID: 1, Status: constants.ContractStatusSigned}
	if err := contracts.AddSigner(&model.ContractSigner{ContractID: 7, Name: "张三", Role: "甲方"}); err != nil {
		t.Fatalf("seed signer: %v", err)
	}

	tickets := newMockTicketRepo()
	resolvable := &model.LegalTicket{
		UserID: 1, Type: constants.TicketTypeContract, Title: "t1",
		Description: "纠纷涉及合同编号 7，对方违约", Status: constants.TicketStatusProcessing,
	}
	noNumber := &model.LegalTicket{
		UserID: 1, Type: constants.TicketTypeOther, Title: "t2",
		Description: "没有提到任何合同", Status: constants.TicketStatusPending,
	}
	notFound := &model.LegalTicket{
		UserID: 1, Type: constants.TicketTypeContract, Title: "t3",
		Description: "合同号 999 找不到", Status: constants.TicketStatusPending,
	}
	for _, tk := range []*model.LegalTicket{resolvable, noNumber, notFound} {
		if err := tickets.Create(tk); err != nil {
			t.Fatalf("seed ticket: %v", err)
		}
	}

	svc := service.NewBackfillService(tickets, contracts, testLogger())
	result, err := svc.Run(nil)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(result.Backfilled) != 1 || result.Backfilled[0].TicketID != resolvable.ID ||
		result.Backfilled[0].ContractID != 7 {
		t.Fatalf("backfilled = %+v, want only ticket %d -> contract 7", result.Backfilled, resolvable.ID)
	}
	if len(result.Unresolved) != 2 {
		t.Fatalf("unresolved = %+v, want 2 tickets listed separately", result.Unresolved)
	}

	stored := tickets.tickets[resolvable.ID]
	if stored.ContractID != 7 || stored.ContractStatusSnapshot != constants.ContractStatusSigned {
		t.Fatalf("ticket snapshot not backfilled: %+v", stored)
	}
	if len(stored.ContractSignersSnapshot) != 1 {
		t.Fatalf("signers snapshot = %d, want 1", len(stored.ContractSignersSnapshot))
	}
	for _, tk := range []*model.LegalTicket{noNumber, notFound} {
		if tickets.tickets[tk.ID].ContractID != 0 {
			t.Fatalf("ticket %d should remain unlinked", tk.ID)
		}
	}

	// 幂等：再跑一次，已回填的不再出现。
	result2, err := svc.Run(nil)
	if err != nil {
		t.Fatalf("Run() second time error: %v", err)
	}
	if len(result2.Backfilled) != 0 || len(result2.Unresolved) != 2 {
		t.Fatalf("second run = %+v, want 0 backfilled / 2 unresolved", result2)
	}
}
