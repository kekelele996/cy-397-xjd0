package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
)

// backfillBatchSize 旧数据回填单批处理上限。
const backfillBatchSize = 200

// TicketService 法律工单提交、流转与回复业务。
type TicketService struct {
	ticketRepo   repository.TicketRepository
	contractRepo repository.ContractRepository
	logger       *slog.Logger
}

// NewTicketService 构造工单服务。
func NewTicketService(
	ticketRepo repository.TicketRepository,
	contractRepo repository.ContractRepository,
	logger *slog.Logger,
) *TicketService {
	return &TicketService{ticketRepo: ticketRepo, contractRepo: contractRepo, logger: logger}
}

// Create 提交法律咨询工单；传入合同编号时校验合同存在并登记当时的合同状态快照。
func (s *TicketService) Create(userID uint64, req dto.CreateTicketRequest) (*model.LegalTicket, error) {
	if !constants.IsValidTicketType(req.Type) {
		return nil, dto.ValidationError("invalid ticket type")
	}
	ticket := &model.LegalTicket{
		UserID:      userID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Attachments: model.StringSlice(req.Attachments),
		Status:      constants.TicketStatusPending,
	}
	if req.ContractNo != "" {
		if !constants.IsValidContractNo(req.ContractNo) {
			return nil, dto.ValidationError("invalid contract_no")
		}
		contract, err := s.contractRepo.FindByContractNo(req.ContractNo)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, dto.NotFoundError("contract not found: " + req.ContractNo)
			}
			return nil, fmt.Errorf("create ticket: find contract: %w", err)
		}
		ticket.ContractNo = contract.ContractNo
		ticket.ContractStatusSnapshot = contract.Status
	}
	if err := s.ticketRepo.Create(ticket); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	s.logger.Info("ticket created", "ticket_id", ticket.ID, "user_id", userID,
		"type", ticket.Type, "contract_no", ticket.ContractNo, "snapshot", ticket.ContractStatusSnapshot)
	return ticket, nil
}

// GetForUser 查询用户自己的工单及回复。
func (s *TicketService) GetForUser(userID, ticketID uint64) (*model.LegalTicket, []model.TicketReply, error) {
	ticket, err := s.ticketRepo.FindByIDForUser(ticketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, dto.NotFoundError("ticket not found")
		}
		return nil, nil, fmt.Errorf("get ticket: %w", err)
	}
	replies, err := s.ticketRepo.ListReplies(ticketID)
	if err != nil {
		return nil, nil, fmt.Errorf("get ticket replies: %w", err)
	}
	return ticket, replies, nil
}

// GetDetailForUser 查询工单详情，并带出合同库此刻的状态与签署方。
func (s *TicketService) GetDetailForUser(userID, ticketID uint64) (*dto.TicketDetailView, error) {
	ticket, replies, err := s.GetForUser(userID, ticketID)
	if err != nil {
		return nil, err
	}
	view := &dto.TicketDetailView{
		Ticket:                   ticket,
		Replies:                  replies,
		RegisteredContractNo:     ticket.ContractNo,
		RegisteredContractStatus: ticket.ContractStatusSnapshot,
	}
	if ticket.ContractNo != "" {
		view.Contract = s.buildContractSnapshot(ticket.ContractNo)
	}
	return view, nil
}

// buildContractSnapshot 读取合同库当前状态与签署方；合同已删除时返回 found=false。
func (s *TicketService) buildContractSnapshot(contractNo string) *dto.ContractSnapshotView {
	snapshot := &dto.ContractSnapshotView{ContractNo: contractNo, Signers: []dto.SignerView{}}
	contract, err := s.contractRepo.FindByContractNo(contractNo)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			s.logger.Error("build contract snapshot: find contract", "contract_no", contractNo, "error", err)
		}
		return snapshot
	}
	snapshot.Found = true
	snapshot.Status = contract.Status
	snapshot.Title = contract.Title
	signers, err := s.contractRepo.ListSigners(contract.ID)
	if err != nil {
		s.logger.Error("build contract snapshot: list signers", "contract_no", contractNo, "error", err)
		return snapshot
	}
	for _, signer := range signers {
		snapshot.Signers = append(snapshot.Signers, dto.SignerView{
			Name:     signer.Name,
			Role:     signer.Role,
			SignInfo: signer.SignInfo,
		})
	}
	return snapshot
}

// ListForUser 查询用户工单列表，支持按状态筛选。
func (s *TicketService) ListForUser(userID uint64, status string, page, pageSize int) ([]model.LegalTicket, int64, error) {
	if status != "" && !constants.IsValidTicketStatus(status) {
		return nil, 0, dto.ValidationError("invalid ticket status")
	}
	offset := (page - 1) * pageSize
	list, total, err := s.ticketRepo.ListByUser(userID, status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list tickets: %w", err)
	}
	return list, total, nil
}

// AddReply 添加工单回复，并按回复角色推进状态。回复仅写工单侧，合同库不参与。
func (s *TicketService) AddReply(userID, ticketID uint64, role, content string, attachments []string) (*model.TicketReply, error) {
	ticket, err := s.ticketRepo.FindByIDForUser(ticketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("ticket not found")
		}
		return nil, fmt.Errorf("add ticket reply: %w", err)
	}
	if ticket.Status == constants.TicketStatusClosed {
		return nil, dto.InvalidTransitionError("cannot reply to closed ticket")
	}
	if ticket.Status == constants.TicketStatusReviewPending {
		return nil, dto.InvalidTransitionError("ticket is suspended pending review; resume to processing before replying")
	}
	if role == "" {
		role = constants.TicketReplyRoleUser
	}
	reply := &model.TicketReply{
		TicketID:    ticket.ID,
		Role:        role,
		Content:     content,
		Attachments: model.StringSlice(attachments),
	}
	// 回复与工单状态推进在工单侧本地事务内完成：失败整批回滚，只重试工单这批即可。
	err = s.ticketRepo.InTx(func(repo repository.TicketRepository) error {
		if err := repo.AddReply(reply); err != nil {
			return fmt.Errorf("add ticket reply: save: %w", err)
		}
		if ticket.Status == constants.TicketStatusPending {
			ticket.Status = constants.TicketStatusProcessing
		}
		if role == constants.TicketReplyRoleLawyer && ticket.Status == constants.TicketStatusProcessing {
			ticket.Status = constants.TicketStatusReplied
		}
		if err := repo.Update(ticket); err != nil {
			return fmt.Errorf("add ticket reply: update ticket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info("ticket reply added", "ticket_id", ticket.ID, "reply_id", reply.ID, "role", role)
	return reply, nil
}

// Transition 手动推进工单状态；关单前强制对账合同库现状。
func (s *TicketService) Transition(userID, ticketID uint64, status string) error {
	ticket, err := s.ticketRepo.FindByIDForUser(ticketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("ticket not found")
		}
		return fmt.Errorf("transition ticket: %w", err)
	}
	if !constants.CanTransitionTicket(ticket.Status, status) {
		return dto.InvalidTransitionError(fmt.Sprintf("cannot transition ticket from %q to %q", ticket.Status, status))
	}
	if status == constants.TicketStatusClosed {
		if mismatch := s.reconcileBeforeClose(ticket); mismatch != nil {
			// 对不上：挂起工单并返回差异明细，禁止关单。
			ticket.Status = constants.TicketStatusReviewPending
			if updateErr := s.ticketRepo.Update(ticket); updateErr != nil {
				return fmt.Errorf("suspend ticket after reconcile mismatch: %w", updateErr)
			}
			s.logger.Warn("ticket close blocked by reconcile", "ticket_id", ticket.ID,
				"contract_no", mismatch.ContractNo, "reason", mismatch.Reason)
			return dto.ReconcileFailedError("ticket suspended: contract status does not match registration",
				[]dto.ReconcileMismatch{*mismatch})
		}
	}
	ticket.Status = status
	if err := s.ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("transition ticket: update: %w", err)
	}
	s.logger.Info("ticket status changed", "ticket_id", ticket.ID, "status", status)
	return nil
}

// reconcileBeforeClose 拿合同库现状与工单登记快照对账；一致返回 nil，不一致返回差异。
func (s *TicketService) reconcileBeforeClose(ticket *model.LegalTicket) *dto.ReconcileMismatch {
	// 旧数据未登记合同编号：无编号可对，不拦截关单（回填异常另行单列）。
	if ticket.ContractNo == "" {
		return nil
	}
	mismatch := &dto.ReconcileMismatch{
		TicketID:         ticket.ID,
		ContractNo:       ticket.ContractNo,
		RegisteredStatus: ticket.ContractStatusSnapshot,
	}
	contract, err := s.contractRepo.FindByContractNo(ticket.ContractNo)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			mismatch.Reason = "contract_not_found"
			return mismatch
		}
		s.logger.Error("reconcile ticket: find contract", "contract_no", ticket.ContractNo, "error", err)
		mismatch.Reason = "contract_query_failed"
		return mismatch
	}
	mismatch.CurrentContractFound = true
	mismatch.CurrentStatus = contract.Status
	if contract.Status != ticket.ContractStatusSnapshot {
		mismatch.Reason = "status_mismatch"
		return mismatch
	}
	return nil
}

// ResumeAfterReview 工单组复核通过：以合同库现状刷新登记快照，并从挂起恢复到处理中。
// 合同已不存在则保持挂起并报错，由人工走异常处理。
func (s *TicketService) ResumeAfterReview(userID, ticketID uint64) error {
	ticket, err := s.ticketRepo.FindByIDForUser(ticketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("ticket not found")
		}
		return fmt.Errorf("resume ticket review: %w", err)
	}
	if ticket.Status != constants.TicketStatusReviewPending {
		return dto.InvalidTransitionError(fmt.Sprintf("ticket in status %q is not pending review", ticket.Status))
	}
	if ticket.ContractNo != "" {
		contract, err := s.contractRepo.FindByContractNo(ticket.ContractNo)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return dto.ConflictError("contract still missing, cannot resume review: " + ticket.ContractNo)
			}
			return fmt.Errorf("resume ticket review: find contract: %w", err)
		}
		ticket.ContractStatusSnapshot = contract.Status
	}
	ticket.Status = constants.TicketStatusProcessing
	if err := s.ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("resume ticket review: update: %w", err)
	}
	s.logger.Info("ticket resumed after review with refreshed snapshot",
		"ticket_id", ticket.ID, "contract_no", ticket.ContractNo, "snapshot", ticket.ContractStatusSnapshot)
	return nil
}

// BatchClose 工单组批量关单：逐笔对账，对得上关单，对不上挂起并列出差异。
func (s *TicketService) BatchClose(userID uint64, ticketIDs []uint64) (*dto.BatchCloseResult, error) {
	result := &dto.BatchCloseResult{Closed: []uint64{}, Suspended: []dto.ReconcileMismatch{}}
	seen := make(map[uint64]struct{}, len(ticketIDs))
	for _, id := range ticketIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		ticket, err := s.ticketRepo.FindByIDForUser(id, userID)
		if err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("batch close: find ticket %d: %w", id, err)
			}
			result.Suspended = append(result.Suspended, dto.ReconcileMismatch{TicketID: id, Reason: "ticket_not_found"})
			continue
		}
		if ticket.Status == constants.TicketStatusClosed {
			result.Closed = append(result.Closed, id)
			continue
		}
		if mismatch := s.reconcileBeforeClose(ticket); mismatch != nil {
			ticket.Status = constants.TicketStatusReviewPending
			if err := s.ticketRepo.Update(ticket); err != nil {
				return nil, fmt.Errorf("batch close: suspend ticket %d: %w", id, err)
			}
			result.Suspended = append(result.Suspended, *mismatch)
			continue
		}
		ticket.Status = constants.TicketStatusClosed
		if err := s.ticketRepo.Update(ticket); err != nil {
			return nil, fmt.Errorf("batch close: close ticket %d: %w", id, err)
		}
		result.Closed = append(result.Closed, id)
	}
	s.logger.Info("batch close tickets", "user_id", userID,
		"closed", len(result.Closed), "suspended", len(result.Suspended))
	return result, nil
}

// BackfillContractNo 对旧数据中未登记合同编号的工单，按问题描述里的编号回填；回填不出的单列。
func (s *TicketService) BackfillContractNo() (*dto.BackfillResult, error) {
	candidates, err := s.ticketRepo.ListBackfillCandidates(backfillBatchSize)
	if err != nil {
		return nil, fmt.Errorf("backfill contract no: list candidates: %w", err)
	}
	result := &dto.BackfillResult{FailedIDs: []uint64{}}
	result.Total = len(candidates)
	for _, ticket := range candidates {
		extracted := constants.ExtractContractNo(ticket.Description)
		if extracted == "" {
			if err := s.ticketRepo.RecordBackfail(ticket.ID, "no_contract_no_in_description", ""); err != nil {
				return nil, fmt.Errorf("backfill contract no: record backfail: %w", err)
			}
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, ticket.ID)
			continue
		}
		contract, err := s.contractRepo.FindByContractNo(extracted)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				if recordErr := s.ticketRepo.RecordBackfail(ticket.ID, "contract_not_found", extracted); recordErr != nil {
					return nil, fmt.Errorf("backfill contract no: record backfail: %w", recordErr)
				}
				result.Failed++
				result.FailedIDs = append(result.FailedIDs, ticket.ID)
				continue
			}
			return nil, fmt.Errorf("backfill contract no: find contract %s: %w", extracted, err)
		}
		if err := s.ticketRepo.AttachContract(ticket.ID, contract.ContractNo, contract.Status); err != nil {
			return nil, fmt.Errorf("backfill contract no: attach ticket %d: %w", ticket.ID, err)
		}
		result.Backfilled++
	}
	s.logger.Info("backfill ticket contract numbers",
		"total", result.Total, "backfilled", result.Backfilled, "failed", result.Failed)
	return result, nil
}

// ListBackfails 查询回填失败的异常清单。
func (s *TicketService) ListBackfails(resolved *bool, page, pageSize int) ([]model.TicketContractBackfail, int64, error) {
	offset := (page - 1) * pageSize
	list, total, err := s.ticketRepo.ListBackfails(resolved, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list backfails: %w", err)
	}
	return list, total, nil
}

// ListReplies 查询工单回复列表。
func (s *TicketService) ListReplies(userID, ticketID uint64) ([]model.TicketReply, error) {
	_, replies, err := s.GetForUser(userID, ticketID)
	return replies, err
}
