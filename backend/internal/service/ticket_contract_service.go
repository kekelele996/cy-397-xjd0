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

// ContractSnapshotReader 读取合同库现状（状态/签署方），供工单登记快照与关单对账使用。
type ContractSnapshotReader interface {
	FindByID(id uint64) (*model.Contract, error)
	FindByIDForUser(id, userID uint64) (*model.Contract, error)
	ListSigners(contractID uint64) ([]model.ContractSigner, error)
}

// TicketSyncer 合同状态落定后联动工单的能力：把该合同下未关工单标成待复核。
// 合同库先落定、后通知工单；工单侧失败不回滚合同，可通过 SyncContractTickets 单独重试这批。
type TicketSyncer interface {
	SyncContractTickets(contractID uint64) (*dto.ContractTicketSyncResult, error)
}

// TicketContractService 工单与合同库联动：登记快照、合同联动、关单对账、复核与旧数据回填。
type TicketContractService struct {
	ticketRepo     repository.TicketRepository
	contractReader ContractSnapshotReader
	logger         *slog.Logger
}

// NewTicketContractService 构造工单-合同联动服务。
func NewTicketContractService(
	ticketRepo repository.TicketRepository,
	contractReader ContractSnapshotReader,
	logger *slog.Logger,
) *TicketContractService {
	return &TicketContractService{ticketRepo: ticketRepo, contractReader: contractReader, logger: logger}
}

// currentContract 读取合同库此刻状态与签署方；合同不存在时返回 nil, nil。
func (s *TicketContractService) currentContract(contractID uint64) (*model.Contract, []model.ContractSigner, error) {
	contract, err := s.contractReader.FindByID(contractID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("load contract %d: %w", contractID, err)
	}
	signers, err := s.contractReader.ListSigners(contractID)
	if err != nil {
		return nil, nil, fmt.Errorf("load contract %d signers: %w", contractID, err)
	}
	return contract, signers, nil
}

// BuildSnapshotView 组装工单详情中的合同库现状视图。
func (s *TicketContractService) BuildSnapshotView(ticket *model.LegalTicket) (*dto.ContractSnapshotView, error) {
	if ticket.ContractID == 0 {
		// 历史遗留、未关联合同的工单无快照可对，视为一致。
		return &dto.ContractSnapshotView{
			ContractID:        0,
			RegisteredSigners: []dto.SignerSnapshotItem{},
			CurrentSigners:    []dto.SignerView{},
			ContractExists:    false,
			Reconciled:        true,
		}, nil
	}

	contract, signers, err := s.currentContract(ticket.ContractID)
	if err != nil {
		return nil, err
	}
	view := &dto.ContractSnapshotView{
		ContractID:        ticket.ContractID,
		RegisteredStatus:  ticket.ContractStatusSnapshot,
		RegisteredSigners: toSnapshotItems(ticket.ContractSignersSnapshot),
		CurrentSigners:    toSignerViews(signers),
	}
	if contract != nil {
		view.ContractExists = true
		view.CurrentStatus = contract.Status
		view.Reconciled = snapshotMatches(ticket, contract, signers)
	} else {
		view.ContractExists = false
		view.Reconciled = false
	}
	return view, nil
}

// CloseWithReconcile 单笔关单：拿合同库现状与工单登记快照对账，对得上才准关；
// 对不上把工单挂起并返回差异明细。
func (s *TicketContractService) CloseWithReconcile(userID, ticketID uint64) (*dto.CloseTicketResult, error) {
	ticket, err := s.ticketRepo.FindByIDForUser(ticketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("ticket not found")
		}
		return nil, fmt.Errorf("close ticket: %w", err)
	}
	if ticket.Status == constants.TicketStatusClosed {
		return &dto.CloseTicketResult{TicketID: ticket.ID, Status: ticket.Status, Closed: true}, nil
	}

	mismatch, err := s.reconcile(ticket)
	if err != nil {
		return nil, err
	}
	if mismatch != nil {
		if err := s.hold(ticket, mismatch); err != nil {
			return nil, err
		}
		return nil, dto.ReconcileMismatchError(
			"contract snapshot mismatch, ticket held",
			map[string]any{"mismatches": []dto.TicketMismatch{*mismatch}},
		)
	}

	ticket.Status = constants.TicketStatusClosed
	if err := s.ticketRepo.Update(ticket); err != nil {
		return nil, fmt.Errorf("close ticket: update: %w", err)
	}
	s.logger.Info("ticket closed after reconciliation", "ticket_id", ticket.ID, "contract_id", ticket.ContractID)
	return &dto.CloseTicketResult{TicketID: ticket.ID, Status: ticket.Status, Closed: true}, nil
}

// BatchClose 批量关单：逐笔对账，关成的与挂起/对不上的分开返回。
func (s *TicketContractService) BatchClose(userID uint64, ticketIDs []uint64) (*dto.BatchCloseTicketResult, error) {
	result := &dto.BatchCloseTicketResult{
		Closed:   []uint64{},
		Mismatch: []dto.TicketMismatch{},
	}
	seen := make(map[uint64]bool, len(ticketIDs))
	for _, id := range ticketIDs {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true

		ticket, err := s.ticketRepo.FindByIDForUser(id, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				result.Mismatch = append(result.Mismatch, dto.TicketMismatch{
					TicketID:    id,
					Reason:      "ticket_not_found",
					Differences: []dto.FieldMismatch{},
				})
				continue
			}
			return nil, fmt.Errorf("batch close ticket %d: %w", id, err)
		}
		if ticket.Status == constants.TicketStatusClosed {
			result.Closed = append(result.Closed, ticket.ID)
			continue
		}

		mismatch, err := s.reconcile(ticket)
		if err != nil {
			return nil, err
		}
		if mismatch != nil {
			if err := s.hold(ticket, mismatch); err != nil {
				return nil, err
			}
			result.Mismatch = append(result.Mismatch, *mismatch)
			continue
		}

		ticket.Status = constants.TicketStatusClosed
		if err := s.ticketRepo.Update(ticket); err != nil {
			return nil, fmt.Errorf("batch close ticket %d: update: %w", id, err)
		}
		result.Closed = append(result.Closed, ticket.ID)
	}
	s.logger.Info("batch ticket close reconciled",
		"closed_count", len(result.Closed), "mismatch_count", len(result.Mismatch))
	return result, nil
}

// Review 工单组复核确认：把待复核/挂起工单登记的合同快照刷新为合同库现状，重新投入处理。
func (s *TicketContractService) Review(userID, ticketID uint64, note string) (*dto.ReviewTicketResult, error) {
	ticket, err := s.ticketRepo.FindByIDForUser(ticketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("ticket not found")
		}
		return nil, fmt.Errorf("review ticket: %w", err)
	}
	if ticket.Status != constants.TicketStatusPendingReview && ticket.Status != constants.TicketStatusOnHold {
		return nil, dto.InvalidTransitionError(
			fmt.Sprintf("only %q/%q tickets can be reviewed, current %q",
				constants.TicketStatusPendingReview, constants.TicketStatusOnHold, ticket.Status))
	}
	if ticket.ContractID == 0 {
		return nil, dto.ValidationError("ticket has no linked contract; nothing to review")
	}

	contract, signers, err := s.currentContract(ticket.ContractID)
	if err != nil {
		return nil, err
	}
	if contract == nil {
		return nil, dto.ConflictError(fmt.Sprintf("contract %d no longer exists", ticket.ContractID))
	}

	ticket.ContractStatusSnapshot = contract.Status
	ticket.ContractSignersSnapshot = model.FromSigners(signers)
	ticket.Status = constants.TicketStatusProcessing
	if err := s.ticketRepo.Update(ticket); err != nil {
		return nil, fmt.Errorf("review ticket: update: %w", err)
	}
	s.logger.Info("ticket reviewed and snapshot refreshed",
		"ticket_id", ticket.ID, "contract_id", ticket.ContractID, "contract_status", contract.Status, "note", note)
	return &dto.ReviewTicketResult{
		TicketID:      ticket.ID,
		Status:        ticket.Status,
		ContractID:    ticket.ContractID,
		CurrentStatus: contract.Status,
	}, nil
}

// SyncContractTickets 合同签署/过期落定后，把该合同下没关的工单标成待复核。
// 已是待复核/挂起的工单保持原状态；返回这批未关工单与实际变更数量。
// 该操作独立于合同落定：失败时只重试本方法，合同状态不受影响。
func (s *TicketContractService) SyncContractTickets(contractID uint64) (*dto.ContractTicketSyncResult, error) {
	open, err := s.ticketRepo.ListOpenByContract(contractID)
	if err != nil {
		return nil, fmt.Errorf("sync contract tickets: list: %w", err)
	}
	affected, err := s.ticketRepo.MarkOpenTicketsPendingReview(contractID)
	if err != nil {
		return nil, fmt.Errorf("sync contract tickets: mark: %w", err)
	}
	ids := make([]uint64, 0, len(open))
	for _, t := range open {
		ids = append(ids, t.ID)
	}
	s.logger.Info("contract tickets marked pending review",
		"contract_id", contractID, "open", len(open), "affected", affected)
	return &dto.ContractTicketSyncResult{
		ContractID:       contractID,
		PendingReviewIDs: ids,
		Affected:         int(affected),
	}, nil
}

// reconcile 拿合同库现状与工单登记快照对账，返回差异；一致时返回 nil。
func (s *TicketContractService) reconcile(ticket *model.LegalTicket) (*dto.TicketMismatch, error) {
	if ticket.ContractID == 0 {
		return nil, nil
	}
	contract, signers, err := s.currentContract(ticket.ContractID)
	if err != nil {
		return nil, err
	}
	diffs := diffSnapshot(ticket, contract, signers)
	if len(diffs) == 0 {
		return nil, nil
	}
	reason := "contract_snapshot_mismatch"
	if contract == nil {
		reason = "contract_not_found"
	}
	return &dto.TicketMismatch{
		TicketID:    ticket.ID,
		ContractID:  ticket.ContractID,
		Reason:      reason,
		Differences: diffs,
	}, nil
}

// hold 对账不一致时先挂起工单。
func (s *TicketContractService) hold(ticket *model.LegalTicket, mismatch *dto.TicketMismatch) error {
	ticket.Status = constants.TicketStatusOnHold
	if err := s.ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("hold ticket %d: %w", ticket.ID, err)
	}
	s.logger.Warn("ticket held due to contract mismatch",
		"ticket_id", ticket.ID, "contract_id", ticket.ContractID, "reason", mismatch.Reason)
	return nil
}

// snapshotMatches 判断工单登记快照与合同库现状是否一致。
func snapshotMatches(ticket *model.LegalTicket, contract *model.Contract, signers []model.ContractSigner) bool {
	return len(diffSnapshot(ticket, contract, signers)) == 0
}

// diffSnapshot 逐字段比对登记快照与现状，列出差异项。
func diffSnapshot(ticket *model.LegalTicket, contract *model.Contract, signers []model.ContractSigner) []dto.FieldMismatch {
	diffs := make([]dto.FieldMismatch, 0, 2)
	if contract == nil {
		diffs = append(diffs, dto.FieldMismatch{
			Field:      "contract",
			Registered: ticket.ContractStatusSnapshot,
			Current:    nil,
		})
		return diffs
	}
	if ticket.ContractStatusSnapshot != contract.Status {
		diffs = append(diffs, dto.FieldMismatch{
			Field:      "contract_status",
			Registered: ticket.ContractStatusSnapshot,
			Current:    contract.Status,
		})
	}
	current := model.FromSigners(signers)
	if !ticket.ContractSignersSnapshot.EqualSet(current) {
		diffs = append(diffs, dto.FieldMismatch{
			Field:      "contract_signers",
			Registered: toSnapshotItems(ticket.ContractSignersSnapshot),
			Current:    toSnapshotItems(current),
		})
	}
	return diffs
}

func toSnapshotItems(in model.SignerSnapshots) []dto.SignerSnapshotItem {
	out := make([]dto.SignerSnapshotItem, 0, len(in))
	for _, s := range in {
		out = append(out, dto.SignerSnapshotItem{Name: s.Name, Role: s.Role})
	}
	return out
}

func toSignerViews(signers []model.ContractSigner) []dto.SignerView {
	out := make([]dto.SignerView, 0, len(signers))
	for _, s := range signers {
		out = append(out, dto.SignerView{
			Name:     s.Name,
			Role:     s.Role,
			SignedAt: s.SignedAt,
			SignInfo: s.SignInfo,
		})
	}
	return out
}
