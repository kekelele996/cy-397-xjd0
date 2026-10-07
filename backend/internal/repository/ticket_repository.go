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
	Update(ticket *model.LegalTicket) error
	AddReply(reply *model.TicketReply) error
	ListReplies(ticketID uint64) ([]model.TicketReply, error)

	// ListOpenByContract 查询某份合同下所有未关工单（合同签署/过期后联动用）。
	ListOpenByContract(contractID uint64) ([]model.LegalTicket, error)
	// MarkOpenTicketsPendingReview 把某份合同下未关工单批量置为待复核。
	// 已是待复核/挂起的工单保持原状态，避免抹掉挂起标记；返回受影响行数。
	MarkOpenTicketsPendingReview(contractID uint64) (int64, error)
	// ListByIDs 按编号批量查询工单。
	ListByIDs(ids []uint64) ([]model.LegalTicket, error)
	// ListWithoutContract 查询历史遗留、未登记合同编号的工单（旧数据回填用）。
	ListWithoutContract(offset, limit int) ([]model.LegalTicket, error)
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

func (r *ticketRepository) ListOpenByContract(contractID uint64) ([]model.LegalTicket, error) {
	var list []model.LegalTicket
	if err := r.db.
		Where("contract_id = ? AND status <> ?", contractID, constants.TicketStatusClosed).
		Order("id ASC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list open tickets by contract: %w", err)
	}
	return list, nil
}

func (r *ticketRepository) MarkOpenTicketsPendingReview(contractID uint64) (int64, error) {
	result := r.db.Model(&model.LegalTicket{}).
		Where("contract_id = ? AND status <> ? AND status <> ? AND status <> ?",
			contractID,
			constants.TicketStatusClosed,
			constants.TicketStatusPendingReview,
			constants.TicketStatusOnHold,
		).
		Update("status", constants.TicketStatusPendingReview)
	if result.Error != nil {
		return 0, fmt.Errorf("mark open tickets pending review: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *ticketRepository) ListByIDs(ids []uint64) ([]model.LegalTicket, error) {
	if len(ids) == 0 {
		return []model.LegalTicket{}, nil
	}
	var list []model.LegalTicket
	if err := r.db.Where("id IN ?", ids).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list tickets by ids: %w", err)
	}
	return list, nil
}

func (r *ticketRepository) ListWithoutContract(offset, limit int) ([]model.LegalTicket, error) {
	var list []model.LegalTicket
	if err := r.db.
		Where("contract_id = 0").
		Order("id ASC").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list tickets without contract: %w", err)
	}
	return list, nil
}
