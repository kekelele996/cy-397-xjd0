package service_test

import (
	"strings"
	"testing"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/service"
)

func newContractFixture() (*mockTemplateRepo, *mockContractRepo, *service.ContractService) {
	templates := newMockTemplateRepo()
	templates.add(&model.ContractTemplate{
		Code:        "lease",
		Name:        "房屋租赁合同",
		Category:    constants.TemplateCategoryLease,
		Content:     "甲方：{{.party_a}}\n乙方：{{.party_b}}\n金额：{{.amount}}",
		ContentHTML: "<p>甲方：{{.party_a}}</p><p>乙方：{{.party_b}}</p><p>金额：{{.amount}}</p>",
		Variables: model.TemplateVariables{
			{Name: "party_a", Label: "甲方", Required: true},
			{Name: "party_b", Label: "乙方", Required: true},
			{Name: "amount", Label: "金额", Required: true},
		},
	})
	contracts := newMockContractRepo()
	svc := service.NewContractService(contracts, templates, service.NewPDFService(testLogger()), testLogger())
	return templates, contracts, svc
}

func TestContractServiceCreate(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]string
		wantErr   bool
		wantText  string
	}{
		{
			name:      "success",
			variables: map[string]string{"party_a": "张三", "party_b": "李四", "amount": "3000"},
			wantErr:   false,
			wantText:  "甲方：张三",
		},
		{
			name:      "missing required",
			variables: map[string]string{"party_a": "张三"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, svc := newContractFixture()
			contract, err := svc.Create(1, dto.CreateContractRequest{
				TemplateID: 1,
				Title:      "测试合同",
				Variables:  tt.variables,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if contract.Status != constants.ContractStatusDraft {
				t.Fatalf("Create() status = %q, want draft", contract.Status)
			}
			if !strings.Contains(contract.ContentText, tt.wantText) {
				t.Fatalf("Create() content = %q, want contains %q", contract.ContentText, tt.wantText)
			}
			if !strings.Contains(contract.ContentHTML, "甲方：张三") {
				t.Fatalf("Create() html = %q, want contains rendered html", contract.ContentHTML)
			}
		})
	}
}

func TestContractServiceSubmitAndSign(t *testing.T) {
	_, _, svc := newContractFixture()
	contract, err := svc.Create(1, dto.CreateContractRequest{
		TemplateID: 1,
		Title:      "签署测试",
		Variables:  map[string]string{"party_a": "张三", "party_b": "李四", "amount": "3000"},
	})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if err := svc.Submit(1, contract.ID, []dto.SignerInput{{Name: "张三", Role: "甲方"}, {Name: "李四", Role: "乙方"}}); err != nil {
		t.Fatalf("Submit() unexpected error: %v", err)
	}
	got, _, err := svc.GetForUser(1, contract.ID)
	if err != nil {
		t.Fatalf("GetForUser() unexpected error: %v", err)
	}
	if got.Status != constants.ContractStatusPendingSign {
		t.Fatalf("status after submit = %q, want pending_signed", got.Status)
	}
	signers, err := svc.ListSigners(1, contract.ID)
	if err != nil || len(signers) != 2 {
		t.Fatalf("ListSigners() = %d, %v; want 2 signers", len(signers), err)
	}

	if err := svc.Sign(1, contract.ID, "张三", "甲方", "本人确认"); err != nil {
		t.Fatalf("Sign() unexpected error: %v", err)
	}
	got, _, err = svc.GetForUser(1, contract.ID)
	if err != nil {
		t.Fatalf("GetForUser() unexpected error: %v", err)
	}
	if got.Status != constants.ContractStatusSigned {
		t.Fatalf("status after sign = %q, want signed", got.Status)
	}
	if got.SignedAt == nil {
		t.Fatal("SignedAt should be recorded after sign")
	}
}

func TestContractServiceInvalidTransition(t *testing.T) {
	_, _, svc := newContractFixture()
	contract, err := svc.Create(1, dto.CreateContractRequest{
		TemplateID: 1,
		Title:      "流转测试",
		Variables:  map[string]string{"party_a": "张三", "party_b": "李四", "amount": "3000"},
	})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if err := svc.Sign(1, contract.ID, "张三", "甲方", ""); err == nil {
		t.Fatal("Sign() expected error when status is draft, got nil")
	}
}
