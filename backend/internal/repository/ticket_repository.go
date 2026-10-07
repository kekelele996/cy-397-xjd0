package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/model"
)

// TicketRepository 法律工单数据访问接口。
type TicketRepository interface {
	Create(ticket *model.LegalTicket) error
	FindByID(id uint64) (*model.LegalTicket, error)
	FindByIDForUser(id, userID uint64) (*model.LegalTicket, error)
	ListByUser(userID uint64, status string, offset, limit int) ([]model.LegalTicket, int64, error)
	ListByIDs(ids []uint64) ([]model.LegalTicket, error)
	Update(ticket *model.LegalTicket) error
	AddReply(reply *model.TicketReply) error
	ListReplies(ticketID uint64) ([]model.TicketReply, error)
	// InTx 在工单库本地事务中执行 fn，失败回滚。
	InTx(fn func(TicketRepository) error) error
	// MarkReviewPendingByContractNo 将该合同编号下所有未关工单置为待复核，返回更新行数。
	MarkReviewPendingByContractNo(contractNo string) (int64, error)
	// ListBackfillCandidates 查询未登记合同编号的历史工单。
	ListBackfillCandidates(limit int) ([]model.LegalTicket, error)
	// AttachContract 回填合同编号与登记时合同状态快照。
	AttachContract(ticketID uint64, contractNo, statusSnapshot string) error
	// RecordBackfail 登记/更新一条回填失败清单。
	RecordBackfail(ticketID uint64, reason, extractedNo string) error
	// ListBackfails 查询回填失败清单。
	ListBackfails(resolved *bool, offset, limit int) ([]model.TicketContractBackfail, int64, error)
}

type ticketRepository struct {
	db *gorm.DB
}

// NewTicketRepository 构造工单仓储。
func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ticket *model.LegalTicket) error {
	if err := r.db.Create(ticket).Error; err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}
	return nil
}

func (r *ticketRepository) FindByID(id uint64) (*model.LegalTicket, error) {
	var ticket model.LegalTicket
	if err := r.db.First(&ticket, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find ticket by id: %w", err)
	}
	return &ticket, nil
}

func (r *ticketRepository) FindByIDForUser(id, userID uint64) (*model.LegalTicket, error) {
	var ticket model.LegalTicket
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find ticket by id and user: %w", err)
	}
	return &ticket, nil
}

func (r *ticketRepository) ListByUser(userID uint64, status string, offset, limit int) ([]model.LegalTicket, int64, error) {
	query := r.db.Model(&model.LegalTicket{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tickets: %w", err)
	}
	var list []model.LegalTicket
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list tickets: %w", err)
	}
	return list, total, nil
}

func (r *ticketRepository) ListByIDs(ids []uint64) ([]model.LegalTicket, error) {
	var list []model.LegalTicket
	if err := r.db.Where("id IN ?", ids).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list tickets by ids: %w", err)
	}
	return list, nil
}

func (r *ticketRepository) Update(ticket *model.LegalTicket) error {
	if err := r.db.Save(ticket).Error; err != nil {
		return fmt.Errorf("update ticket: %w", err)
	}
	return nil
}

func (r *ticketRepository) AddReply(reply *model.TicketReply) error {
	if err := r.db.Create(reply).Error; err != nil {
		return fmt.Errorf("add ticket reply: %w", err)
	}
	return nil
}

func (r *ticketRepository) ListReplies(ticketID uint64) ([]model.TicketReply, error) {
	var list []model.TicketReply
	if err := r.db.Where("ticket_id = ?", ticketID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list ticket replies: %w", err)
	}
	return list, nil
}

func (r *ticketRepository) InTx(fn func(TicketRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&ticketRepository{db: tx})
	})
}

func (r *ticketRepository) MarkReviewPendingByContractNo(contractNo string) (int64, error) {
	result := r.db.Model(&model.LegalTicket{}).
		Where("contract_no = ? AND status <> ?", contractNo, constants.TicketStatusClosed).
		Update("status", constants.TicketStatusReviewPending)
	if result.Error != nil {
		return 0, fmt.Errorf("mark tickets review pending: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *ticketRepository) ListBackfillCandidates(limit int) ([]model.LegalTicket, error) {
	var list []model.LegalTicket
	if err := r.db.Where("contract_no = '' OR contract_no IS NULL").
		Order("id ASC").Limit(limit).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list backfill candidates: %w", err)
	}
	return list, nil
}

func (r *ticketRepository) AttachContract(ticketID uint64, contractNo, statusSnapshot string) error {
	result := r.db.Model(&model.LegalTicket{}).
		Where("id = ?", ticketID).
		Updates(map[string]any{
			"contract_no":              contractNo,
			"contract_status_snapshot": statusSnapshot,
		})
	if result.Error != nil {
		return fmt.Errorf("attach contract to ticket: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ticketRepository) RecordBackfail(ticketID uint64, reason, extractedNo string) error {
	var backfail model.TicketContractBackfail
	err := r.db.Where("ticket_id = ?", ticketID).First(&backfail).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		backfail = model.TicketContractBackfail{
			TicketID:    ticketID,
			Reason:      reason,
			ExtractedNo: extractedNo,
			Resolved:    false,
		}
		if createErr := r.db.Create(&backfail).Error; createErr != nil {
			return fmt.Errorf("record backfail: %w", createErr)
		}
	case err != nil:
		return fmt.Errorf("find backfail: %w", err)
	default:
		backfail.Reason = reason
		backfail.ExtractedNo = extractedNo
		backfail.Resolved = false
		if saveErr := r.db.Save(&backfail).Error; saveErr != nil {
			return fmt.Errorf("update backfail: %w", saveErr)
		}
	}
	return nil
}

func (r *ticketRepository) ListBackfails(resolved *bool, offset, limit int) ([]model.TicketContractBackfail, int64, error) {
	query := r.db.Model(&model.TicketContractBackfail{})
	if resolved != nil {
		query = query.Where("resolved = ?", *resolved)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count backfails: %w", err)
	}
	var list []model.TicketContractBackfail
	if err := query.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list backfails: %w", err)
	}
	return list, total, nil
}
