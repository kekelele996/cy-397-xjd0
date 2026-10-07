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

// TicketService 法律工单提交、流转与回复业务。
type TicketService struct {
	ticketRepo repository.TicketRepository
	logger     *slog.Logger
}

// NewTicketService 构造工单服务。
func NewTicketService(ticketRepo repository.TicketRepository, logger *slog.Logger) *TicketService {
	return &TicketService{ticketRepo: ticketRepo, logger: logger}
}

// Create 提交法律咨询工单。
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
	if err := s.ticketRepo.Create(ticket); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	s.logger.Info("ticket created", "ticket_id", ticket.ID, "user_id", userID, "type", ticket.Type)
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

// ListForUser 查询用户工单列表，支持按状态筛选。
func (s *TicketService) ListForUser(userID uint64, status string, page, pageSize int) ([]model.LegalTicket, int64, error) {
	if status != "" && !isValidTicketStatus(status) {
		return nil, 0, dto.ValidationError("invalid ticket status")
	}
	offset := (page - 1) * pageSize
	list, total, err := s.ticketRepo.ListByUser(userID, status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list tickets: %w", err)
	}
	return list, total, nil
}

// AddReply 添加工单回复，并按回复角色推进状态。
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
	if role == "" {
		role = constants.TicketReplyRoleUser
	}
	reply := &model.TicketReply{
		TicketID:    ticket.ID,
		Role:        role,
		Content:     content,
		Attachments: model.StringSlice(attachments),
	}
	if err := s.ticketRepo.AddReply(reply); err != nil {
		return nil, fmt.Errorf("add ticket reply: save: %w", err)
	}
	if ticket.Status == constants.TicketStatusPending {
		ticket.Status = constants.TicketStatusProcessing
	}
	if role == constants.TicketReplyRoleLawyer && ticket.Status == constants.TicketStatusProcessing {
		ticket.Status = constants.TicketStatusReplied
	}
	if ticket.Status != "" {
		if err := s.ticketRepo.Update(ticket); err != nil {
			return nil, fmt.Errorf("add ticket reply: update ticket: %w", err)
		}
	}
	s.logger.Info("ticket reply added", "ticket_id", ticket.ID, "reply_id", reply.ID, "role", role)
	return reply, nil
}

// Transition 手动推进工单状态。
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
	ticket.Status = status
	if err := s.ticketRepo.Update(ticket); err != nil {
		return fmt.Errorf("transition ticket: update: %w", err)
	}
	s.logger.Info("ticket status changed", "ticket_id", ticket.ID, "status", status)
	return nil
}

// ListReplies 查询工单回复列表。
func (s *TicketService) ListReplies(userID, ticketID uint64) ([]model.TicketReply, error) {
	_, replies, err := s.GetForUser(userID, ticketID)
	return replies, err
}

func isValidTicketStatus(status string) bool {
	switch status {
	case constants.TicketStatusPending, constants.TicketStatusProcessing,
		constants.TicketStatusReplied, constants.TicketStatusClosed:
		return true
	default:
		return false
	}
}
