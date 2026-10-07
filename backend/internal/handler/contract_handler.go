package handler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/middleware"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/service"
)

// ContractHandler 合同生成、签署与导出处理。
type ContractHandler struct {
	contractService *service.ContractService
	logger          *slog.Logger
}

// NewContractHandler 构造合同处理器。
func NewContractHandler(contractService *service.ContractService, logger *slog.Logger) *ContractHandler {
	return &ContractHandler{contractService: contractService, logger: logger}
}

// Create POST /api/v1/contracts
func (h *ContractHandler) Create(c *gin.Context) {
	var req dto.CreateContractRequest
	if !bindJSON(c, &req) {
		return
	}
	contract, err := h.contractService.Create(middleware.CurrentUserID(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, contract)
}

// List GET /api/v1/contracts
func (h *ContractHandler) List(c *gin.Context) {
	var req dto.ContractListQuery
	if !bindQuery(c, &req) {
		return
	}
	page, pageSize := req.Pagination.Normalize()
	list, total, err := h.contractService.ListForUser(middleware.CurrentUserID(c), req.Status, page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}
	dto.SuccessPage(c, list, total, page, pageSize)
}

// Get GET /api/v1/contracts/:id
func (h *ContractHandler) Get(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	contract, templateName, err := h.contractService.GetForUser(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, buildContractView(contract, templateName))
}

// Submit POST /api/v1/contracts/:id/submit
func (h *ContractHandler) Submit(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.SubmitContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(dto.ValidationError(err.Error()))
		return
	}
	if err := h.contractService.Submit(middleware.CurrentUserID(c), id, req.Signers); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"contract_id": id, "status": constants.ContractStatusPendingSign})
}

// Sign POST /api/v1/contracts/:id/sign
func (h *ContractHandler) Sign(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.SignContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(dto.ValidationError(err.Error()))
		return
	}
	signerName := req.SignerName
	if signerName == "" {
		signerName = middleware.CurrentUsername(c)
	}
	if err := h.contractService.Sign(middleware.CurrentUserID(c), id, signerName, req.SignerRole, req.SignInfo); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"contract_id": id, "status": constants.ContractStatusSigned})
}

// Expire POST /api/v1/contracts/:id/expire
func (h *ContractHandler) Expire(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.contractService.Expire(middleware.CurrentUserID(c), id); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"contract_id": id, "status": constants.ContractStatusExpired})
}

// Signers GET /api/v1/contracts/:id/signers
func (h *ContractHandler) Signers(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	signers, err := h.contractService.ListSigners(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, signers)
}

// Export GET /api/v1/contracts/:id/export
func (h *ContractHandler) Export(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	path, err := h.contractService.ExportPDF(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	defer h.contractService.CleanupPDF(path)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"contract-%d.pdf\"", id))
	c.File(path)
}

func buildContractView(contract *model.Contract, templateName string) dto.ContractView {
	return dto.ContractView{
		ID:           contract.ID,
		UserID:       contract.UserID,
		TemplateID:   contract.TemplateID,
		TemplateName: templateName,
		Title:        contract.Title,
		ContentText:  contract.ContentText,
		ContentHTML:  contract.ContentHTML,
		Status:       contract.Status,
		Variables:    map[string]string(contract.Variables),
		SignedAt:     contract.SignedAt,
		ExpiresAt:    contract.ExpiresAt,
		CreatedAt:    contract.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    contract.UpdatedAt.Format(time.RFC3339),
	}
}
