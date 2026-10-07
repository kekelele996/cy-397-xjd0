package service_test

import (
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
)

type mockUserRepo struct {
	users       map[string]*model.User
	nextID      uint64
	createErr   error
	usernameErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: map[string]*model.User{}, nextID: 1}
}

func (m *mockUserRepo) Create(user *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, ok := m.users[user.Username]; ok {
		return repository.ErrDuplicateKey
	}
	user.ID = m.nextID
	m.nextID++
	m.users[user.Username] = user
	return nil
}

func (m *mockUserRepo) FindByUsername(username string) (*model.User, error) {
	if m.usernameErr != nil {
		return nil, m.usernameErr
	}
	user, ok := m.users[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (m *mockUserRepo) FindByID(id uint64) (*model.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

type mockTemplateRepo struct {
	templates map[uint64]*model.ContractTemplate
	nextID    uint64
}

func newMockTemplateRepo() *mockTemplateRepo {
	return &mockTemplateRepo{templates: map[uint64]*model.ContractTemplate{}, nextID: 1}
}

func (m *mockTemplateRepo) add(template *model.ContractTemplate) *model.ContractTemplate {
	template.ID = m.nextID
	m.nextID++
	m.templates[template.ID] = template
	return template
}

func (m *mockTemplateRepo) List(category, keyword string, offset, limit int) ([]model.ContractTemplate, int64, error) {
	var list []model.ContractTemplate
	for _, item := range m.templates {
		list = append(list, *item)
	}
	total := int64(len(list))
	if offset > len(list) {
		offset = len(list)
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], total, nil
}

func (m *mockTemplateRepo) FindByID(id uint64) (*model.ContractTemplate, error) {
	item, ok := m.templates[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return item, nil
}

func (m *mockTemplateRepo) Create(template *model.ContractTemplate) error {
	m.add(template)
	return nil
}

func (m *mockTemplateRepo) Update(template *model.ContractTemplate) error {
	if _, ok := m.templates[template.ID]; !ok {
		return repository.ErrNotFound
	}
	m.templates[template.ID] = template
	return nil
}

func (m *mockTemplateRepo) Delete(id uint64) error {
	delete(m.templates, id)
	return nil
}

type mockFavoriteRepo struct {
	favorites map[uint64]map[uint64]bool
}

func newMockFavoriteRepo() *mockFavoriteRepo {
	return &mockFavoriteRepo{favorites: map[uint64]map[uint64]bool{}}
}

func (m *mockFavoriteRepo) Create(userID, templateID uint64) error {
	if m.favorites[userID] == nil {
		m.favorites[userID] = map[uint64]bool{}
	}
	if m.favorites[userID][templateID] {
		return repository.ErrDuplicateKey
	}
	m.favorites[userID][templateID] = true
	return nil
}

func (m *mockFavoriteRepo) Delete(userID, templateID uint64) error {
	if m.favorites[userID] == nil || !m.favorites[userID][templateID] {
		return repository.ErrNotFound
	}
	delete(m.favorites[userID], templateID)
	return nil
}

func (m *mockFavoriteRepo) Exists(userID, templateID uint64) (bool, error) {
	return m.favorites[userID][templateID], nil
}

func (m *mockFavoriteRepo) ListByUser(userID uint64, offset, limit int) ([]model.ContractTemplate, int64, error) {
	return nil, 0, nil
}

type mockContractRepo struct {
	contracts map[uint64]*model.Contract
	signers   map[uint64][]model.ContractSigner
	nextID    uint64
}

func newMockContractRepo() *mockContractRepo {
	return &mockContractRepo{contracts: map[uint64]*model.Contract{}, signers: map[uint64][]model.ContractSigner{}, nextID: 1}
}

func (m *mockContractRepo) Create(contract *model.Contract) error {
	contract.ID = m.nextID
	m.nextID++
	m.contracts[contract.ID] = contract
	return nil
}

func (m *mockContractRepo) FindByID(id uint64) (*model.Contract, error) {
	contract, ok := m.contracts[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return contract, nil
}

func (m *mockContractRepo) FindByIDForUser(id, userID uint64) (*model.Contract, error) {
	contract, ok := m.contracts[id]
	if !ok || contract.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return contract, nil
}

func (m *mockContractRepo) ListByUser(userID uint64, status string, offset, limit int) ([]model.Contract, int64, error) {
	var list []model.Contract
	for _, item := range m.contracts {
		if item.UserID != userID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		list = append(list, *item)
	}
	return list, int64(len(list)), nil
}

func (m *mockContractRepo) Update(contract *model.Contract) error {
	if _, ok := m.contracts[contract.ID]; !ok {
		return repository.ErrNotFound
	}
	m.contracts[contract.ID] = contract
	return nil
}

func (m *mockContractRepo) AddSigner(signer *model.ContractSigner) error {
	m.signers[signer.ContractID] = append(m.signers[signer.ContractID], *signer)
	return nil
}

func (m *mockContractRepo) ListSigners(contractID uint64) ([]model.ContractSigner, error) {
	return m.signers[contractID], nil
}

type mockTicketRepo struct {
	tickets map[uint64]*model.LegalTicket
	replies map[uint64][]model.TicketReply
	nextID  uint64
}

func newMockTicketRepo() *mockTicketRepo {
	return &mockTicketRepo{tickets: map[uint64]*model.LegalTicket{}, replies: map[uint64][]model.TicketReply{}, nextID: 1}
}

func (m *mockTicketRepo) Create(ticket *model.LegalTicket) error {
	ticket.ID = m.nextID
	m.nextID++
	m.tickets[ticket.ID] = ticket
	return nil
}

func (m *mockTicketRepo) FindByID(id uint64) (*model.LegalTicket, error) {
	ticket, ok := m.tickets[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return ticket, nil
}

func (m *mockTicketRepo) FindByIDForUser(id, userID uint64) (*model.LegalTicket, error) {
	ticket, ok := m.tickets[id]
	if !ok || ticket.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return ticket, nil
}

func (m *mockTicketRepo) ListByUser(userID uint64, status string, offset, limit int) ([]model.LegalTicket, int64, error) {
	var list []model.LegalTicket
	for _, item := range m.tickets {
		if item.UserID != userID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		list = append(list, *item)
	}
	return list, int64(len(list)), nil
}

func (m *mockTicketRepo) Update(ticket *model.LegalTicket) error {
	if _, ok := m.tickets[ticket.ID]; !ok {
		return repository.ErrNotFound
	}
	m.tickets[ticket.ID] = ticket
	return nil
}

func (m *mockTicketRepo) AddReply(reply *model.TicketReply) error {
	reply.ID = uint64(len(m.replies[reply.TicketID]) + 1)
	m.replies[reply.TicketID] = append(m.replies[reply.TicketID], *reply)
	return nil
}

func (m *mockTicketRepo) ListReplies(ticketID uint64) ([]model.TicketReply, error) {
	return m.replies[ticketID], nil
}

