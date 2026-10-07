package service

import (
	"bytes"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
)

// ContractSettleResult 合同签署/过期落定结果，含关联工单待复核联动情况。
type ContractSettleResult struct {
	ContractID      uint64 `json:"contract_id"`
	ContractNo      string `json:"contract_no"`
	Status          string `json:"status"`
	ReviewMarked    int64  `json:"review_marked"`
	ReviewSyncError bool   `json:"review_sync_error"`
}

// ContractService 合同生成、签署状态流转与导出业务。
type ContractService struct {
	contractRepo repository.ContractRepository
	ticketRepo   repository.TicketRepository
	templateRepo repository.TemplateRepository
	pdf          *PDFService
	logger       *slog.Logger
}

// NewContractService 构造合同服务。
func NewContractService(
	contractRepo repository.ContractRepository,
	ticketRepo repository.TicketRepository,
	templateRepo repository.TemplateRepository,
	pdf *PDFService,
	logger *slog.Logger,
) *ContractService {
	return &ContractService{
		contractRepo: contractRepo,
		ticketRepo:   ticketRepo,
		templateRepo: templateRepo,
		pdf:          pdf,
		logger:       logger,
	}
}

// buildContractNo 生成合同业务编号：HT-年份-6 位主键序号。
func buildContractNo(id uint64, createdAt time.Time) string {
	return fmt.Sprintf("HT-%d-%06d", createdAt.Year(), id)
}

// Create 根据模板与变量生成合同草稿。
func (s *ContractService) Create(userID uint64, req dto.CreateContractRequest) (*model.Contract, error) {
	templateModel, err := s.templateRepo.FindByID(req.TemplateID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("template not found")
		}
		return nil, fmt.Errorf("create contract: find template: %w", err)
	}
	variables := req.Variables
	if variables == nil {
		variables = map[string]string{}
	}
	missing := missingRequiredVariables(templateModel.Variables, variables)
	if len(missing) > 0 {
		return nil, dto.ValidationError("missing required variables: " + strings.Join(missing, ", "))
	}
	contentText, err := renderTextContract(templateModel.Content, variables)
	if err != nil {
		return nil, fmt.Errorf("create contract: render text: %w", err)
	}
	contentHTML, err := renderHTMLContract(templateModel.ContentHTML, contentText, variables)
	if err != nil {
		return nil, fmt.Errorf("create contract: render html: %w", err)
	}
	contract := &model.Contract{
		UserID:      userID,
		TemplateID:  templateModel.ID,
		Title:       req.Title,
		ContentText: contentText,
		ContentHTML: contentHTML,
		Status:      constants.ContractStatusDraft,
		Variables:   model.JSONMap(variables),
	}
	if err := s.contractRepo.Create(contract); err != nil {
		return nil, fmt.Errorf("create contract: save: %w", err)
	}
	// 以主键序号生成业务合同编号，并作为工单侧对账关联键。
	if contract.ContractNo == "" {
		contract.ContractNo = buildContractNo(contract.ID, contract.CreatedAt)
		if err := s.contractRepo.Update(contract); err != nil {
			return nil, fmt.Errorf("create contract: assign contract_no: %w", err)
		}
	}
	s.logger.Info("contract created", "contract_id", contract.ID, "contract_no", contract.ContractNo,
		"user_id", userID, "template_id", templateModel.ID)
	return contract, nil
}

// GetForUser 查询用户自己的合同，并返回模板名称。
func (s *ContractService) GetForUser(userID, contractID uint64) (*model.Contract, string, error) {
	contract, err := s.contractRepo.FindByIDForUser(contractID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", dto.NotFoundError("contract not found")
		}
		return nil, "", fmt.Errorf("get contract: %w", err)
	}
	templateName := ""
	if templateModel, err := s.templateRepo.FindByID(contract.TemplateID); err == nil {
		templateName = templateModel.Name
	}
	return contract, templateName, nil
}

// ListForUser 查询用户合同库，支持按状态筛选。
func (s *ContractService) ListForUser(userID uint64, status string, page, pageSize int) ([]model.Contract, int64, error) {
	if status != "" && !constants.IsValidContractStatus(status) {
		return nil, 0, dto.ValidationError("invalid contract status")
	}
	offset := (page - 1) * pageSize
	list, total, err := s.contractRepo.ListByUser(userID, status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list contracts: %w", err)
	}
	return list, total, nil
}

// Submit 提交合同进入待签署状态，并记录签署方信息。
func (s *ContractService) Submit(userID, contractID uint64, signers []dto.SignerInput) error {
	contract, err := s.contractRepo.FindByIDForUser(contractID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("contract not found")
		}
		return fmt.Errorf("submit contract: %w", err)
	}
	if !constants.CanTransitionContract(contract.Status, constants.ContractStatusPendingSign) {
		return dto.InvalidTransitionError(fmt.Sprintf("cannot submit contract in status %q", contract.Status))
	}
	contract.Status = constants.ContractStatusPendingSign
	if err := s.contractRepo.Update(contract); err != nil {
		return fmt.Errorf("submit contract: update: %w", err)
	}
	for _, signer := range signers {
		if err := s.contractRepo.AddSigner(&model.ContractSigner{
			ContractID: contract.ID,
			Name:       signer.Name,
			Role:       signer.Role,
		}); err != nil {
			return fmt.Errorf("submit contract: add signer: %w", err)
		}
	}
	s.logger.Info("contract submitted", "contract_id", contract.ID)
	return nil
}

// Sign 将待签署合同置为已签署，记录签署时间和签署方；合同库落定后联动该编号下未关工单待复核。
func (s *ContractService) Sign(userID, contractID uint64, signerName, signerRole, signInfo string) (*ContractSettleResult, error) {
	contract, err := s.contractRepo.FindByIDForUser(contractID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("contract not found")
		}
		return nil, fmt.Errorf("sign contract: %w", err)
	}
	if !constants.CanTransitionContract(contract.Status, constants.ContractStatusSigned) {
		return nil, dto.InvalidTransitionError(fmt.Sprintf("cannot sign contract in status %q", contract.Status))
	}
	now := time.Now()
	contract.Status = constants.ContractStatusSigned
	contract.SignedAt = &now
	if err := s.contractRepo.Update(contract); err != nil {
		return nil, fmt.Errorf("sign contract: update: %w", err)
	}
	if signerRole == "" {
		signerRole = "签署方"
	}
	if err := s.contractRepo.AddSigner(&model.ContractSigner{
		ContractID: contract.ID,
		Name:       signerName,
		Role:       signerRole,
		SignedAt:   &now,
		SignInfo:   signInfo,
	}); err != nil {
		return nil, fmt.Errorf("sign contract: add signer: %w", err)
	}
	s.logger.Info("contract signed", "contract_id", contract.ID, "contract_no", contract.ContractNo, "signer", signerName)
	return s.markTicketsReviewPending(contract), nil
}

// Expire 将合同置为已过期；合同库落定后联动该编号下未关工单待复核。
func (s *ContractService) Expire(userID, contractID uint64) (*ContractSettleResult, error) {
	contract, err := s.contractRepo.FindByIDForUser(contractID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("contract not found")
		}
		return nil, fmt.Errorf("expire contract: %w", err)
	}
	if !constants.CanTransitionContract(contract.Status, constants.ContractStatusExpired) {
		return nil, dto.InvalidTransitionError(fmt.Sprintf("cannot expire contract in status %q", contract.Status))
	}
	now := time.Now()
	contract.Status = constants.ContractStatusExpired
	contract.ExpiresAt = &now
	if err := s.contractRepo.Update(contract); err != nil {
		return nil, fmt.Errorf("expire contract: update: %w", err)
	}
	s.logger.Info("contract expired", "contract_id", contract.ID, "contract_no", contract.ContractNo)
	return s.markTicketsReviewPending(contract), nil
}

// markTicketsReviewPending 合同落定后把该编号下未关工单标成待复核。
// 合同库已落定，不参与回滚；工单批失败只在结果中标记，由 RecheckContractTickets 单独重试。
func (s *ContractService) markTicketsReviewPending(contract *model.Contract) *ContractSettleResult {
	result := &ContractSettleResult{
		ContractID: contract.ID,
		ContractNo: contract.ContractNo,
		Status:     contract.Status,
	}
	if contract.ContractNo == "" {
		return result
	}
	marked, err := s.ticketRepo.MarkReviewPendingByContractNo(contract.ContractNo)
	if err != nil {
		result.ReviewSyncError = true
		s.logger.Error("mark linked tickets review pending failed; contract already settled, retry ticket batch only",
			"contract_no", contract.ContractNo, "error", err)
		return result
	}
	result.ReviewMarked = marked
	s.logger.Info("linked tickets marked review pending", "contract_no", contract.ContractNo, "count", marked)
	return result
}

// RecheckContractTickets 合同已落定但工单批联动失败时，只重试工单这一批；合同库照旧不改动。
func (s *ContractService) RecheckContractTickets(userID, contractID uint64) (*ContractSettleResult, error) {
	contract, err := s.contractRepo.FindByIDForUser(contractID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("contract not found")
		}
		return nil, fmt.Errorf("recheck contract tickets: %w", err)
	}
	return s.markTicketsReviewPending(contract), nil
}

// ListSigners 查询合同签署方。
func (s *ContractService) ListSigners(userID, contractID uint64) ([]model.ContractSigner, error) {
	if _, err := s.contractRepo.FindByIDForUser(contractID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("contract not found")
		}
		return nil, fmt.Errorf("list signers: %w", err)
	}
	signers, err := s.contractRepo.ListSigners(contractID)
	if err != nil {
		return nil, fmt.Errorf("list signers: %w", err)
	}
	return signers, nil
}

// CleanupPDF 删除导出的临时 PDF 文件。
func (s *ContractService) CleanupPDF(path string) {
	s.pdf.Cleanup(path)
}

// ExportPDF 将合同 HTML 导出为 PDF 文件路径。
func (s *ContractService) ExportPDF(userID, contractID uint64) (string, error) {
	contract, _, err := s.GetForUser(userID, contractID)
	if err != nil {
		return "", err
	}
	path, err := s.pdf.Generate(contract.ContentHTML)
	if err != nil {
		return "", fmt.Errorf("export contract pdf: %w", err)
	}
	return path, nil
}

func missingRequiredVariables(variables model.TemplateVariables, values map[string]string) []string {
	missing := make([]string, 0)
	for _, v := range variables {
		if !v.Required {
			continue
		}
		if strings.TrimSpace(values[v.Name]) == "" {
			missing = append(missing, v.Name)
		}
	}
	return missing
}

func renderTextContract(content string, values map[string]string) (string, error) {
	tmpl, err := texttemplate.New("contract").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse text template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return "", fmt.Errorf("execute text template: %w", err)
	}
	return buf.String(), nil
}

func renderHTMLContract(contentHTML, contentText string, values map[string]string) (string, error) {
	if contentHTML == "" {
		escaped := htmltemplate.HTMLEscapeString(contentText)
		return "<html><head><meta charset=\"utf-8\"><title>合同</title></head><body><pre>" + escaped + "</pre></body></html>", nil
	}
	tmpl, err := htmltemplate.New("contract").Parse(contentHTML)
	if err != nil {
		return "", fmt.Errorf("parse html template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return "", fmt.Errorf("execute html template: %w", err)
	}
	return buf.String(), nil
}
