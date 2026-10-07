package service

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
)

// backfillBatchSize 旧数据回填单批扫描上限。
const backfillBatchSize = 500

// 合同编号在问题描述中的常见写法：
//   - 「合同编号 123」「合同号：123」「合同#123」「编号 HT-2026-12」等；
//   - 英文 contract no/id；
//   - 兜底识别「#数字」。
//
// 编号字符集为字母、数字、连字符与下划线；纯日期（如 2026-01-01）不兜底匹配。
var (
	labeledContractNoPattern = regexp.MustCompile(
		`(?:合同编号|合同号|合同编码|编号|contract\s*(?:no\.?|id)|contract)\s*[:：#]?\s*([A-Za-z0-9][A-Za-z0-9_-]{0,31})`)
	hashNumberPattern = regexp.MustCompile(`#\s*(\d{1,18})`)
	datePattern       = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}$`)
)

// BackfillService 旧工单合同编号回填。
type BackfillService struct {
	ticketRepo     repository.TicketRepository
	contractReader ContractSnapshotReader
	logger         *slog.Logger
}

// NewBackfillService 构造回填服务。
func NewBackfillService(
	ticketRepo repository.TicketRepository,
	contractReader ContractSnapshotReader,
	logger *slog.Logger,
) *BackfillService {
	return &BackfillService{ticketRepo: ticketRepo, contractReader: contractReader, logger: logger}
}

// ExtractContractNo 从工单问题描述中提取合同编号。
// 优先匹配「合同编号/合同号/编号」等带标签写法，其次兜底「#数字」；提取不出返回空串。
func ExtractContractNo(description string) string {
	if m := labeledContractNoPattern.FindStringSubmatch(description); len(m) == 2 {
		if !datePattern.MatchString(m[1]) {
			return m[1]
		}
	}
	if m := hashNumberPattern.FindStringSubmatch(description); len(m) == 2 {
		return m[1]
	}
	return ""
}

// Run 扫描未登记合同编号的旧工单：按问题描述中的编号回填，回填不出的单列。
// ticketIDs 非空时只处理指定工单。
func (s *BackfillService) Run(ticketIDs []uint64) (*dto.BackfillTicketResult, error) {
	result := &dto.BackfillTicketResult{
		Backfilled: []dto.BackfilledTicket{},
		Unresolved: []dto.UnresolvedTicket{},
	}

	tickets, err := s.loadTargets(ticketIDs)
	if err != nil {
		return nil, err
	}
	for i := range tickets {
		ticket := &tickets[i]
		if ticket.ContractID != 0 {
			continue
		}
		raw := ExtractContractNo(ticket.Description)
		if raw == "" {
			result.Unresolved = append(result.Unresolved, dto.UnresolvedTicket{
				TicketID: ticket.ID, Reason: "no_contract_no_in_description",
			})
			continue
		}
		contractID, parseErr := parseContractID(raw)
		if parseErr != nil {
			result.Unresolved = append(result.Unresolved, dto.UnresolvedTicket{
				TicketID: ticket.ID, Reason: "contract_no_not_numeric: " + raw,
			})
			continue
		}
		contract, signers, err := s.loadContract(contractID)
		if err != nil {
			return nil, err
		}
		if contract == nil {
			result.Unresolved = append(result.Unresolved, dto.UnresolvedTicket{
				TicketID: ticket.ID, Reason: "contract_not_found: " + raw,
			})
			continue
		}
		ticket.ContractID = contract.ID
		ticket.ContractStatusSnapshot = contract.Status
		ticket.ContractSignersSnapshot = model.FromSigners(signers)
		if err := s.ticketRepo.Update(ticket); err != nil {
			return nil, fmt.Errorf("backfill ticket %d: %w", ticket.ID, err)
		}
		result.Backfilled = append(result.Backfilled, dto.BackfilledTicket{
			TicketID: ticket.ID, ContractID: contract.ID,
		})
	}
	s.logger.Info("ticket contract backfill finished",
		"backfilled", len(result.Backfilled), "unresolved", len(result.Unresolved))
	return result, nil
}

func (s *BackfillService) loadTargets(ticketIDs []uint64) ([]model.LegalTicket, error) {
	if len(ticketIDs) > 0 {
		list, err := s.ticketRepo.ListByIDs(ticketIDs)
		if err != nil {
			return nil, fmt.Errorf("backfill: list target tickets: %w", err)
		}
		return list, nil
	}
	all := make([]model.LegalTicket, 0)
	offset := 0
	for {
		batch, err := s.ticketRepo.ListWithoutContract(offset, backfillBatchSize)
		if err != nil {
			return nil, fmt.Errorf("backfill: list tickets without contract: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < backfillBatchSize {
			break
		}
		offset += backfillBatchSize
	}
	return all, nil
}

func (s *BackfillService) loadContract(id uint64) (*model.Contract, []model.ContractSigner, error) {
	contract, err := s.contractReader.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("backfill: find contract %d: %w", id, err)
	}
	signers, err := s.contractReader.ListSigners(id)
	if err != nil {
		return nil, nil, fmt.Errorf("backfill: list contract %d signers: %w", id, err)
	}
	return contract, signers, nil
}
