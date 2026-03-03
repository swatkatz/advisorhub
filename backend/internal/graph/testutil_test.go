package graph

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/contribution"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
	"github.com/swatkatz/advisorhub/backend/internal/temporal"
	"github.com/swatkatz/advisorhub/backend/internal/transfer"
)

// --- Mock ClientRepository ---

type mockClientRepo struct {
	clients    map[string]*client.Client
	byAdvisor  map[string][]client.Client
	byHousehold map[string][]client.Client
}

func newMockClientRepo() *mockClientRepo {
	return &mockClientRepo{
		clients:     make(map[string]*client.Client),
		byAdvisor:   make(map[string][]client.Client),
		byHousehold: make(map[string][]client.Client),
	}
}

func (m *mockClientRepo) GetClient(_ context.Context, id string) (*client.Client, error) {
	c, ok := m.clients[id]
	if !ok {
		return nil, errors.New("client not found")
	}
	return c, nil
}

func (m *mockClientRepo) GetClients(_ context.Context, advisorID string) ([]client.Client, error) {
	return m.byAdvisor[advisorID], nil
}

func (m *mockClientRepo) GetClientsByHouseholdID(_ context.Context, householdID string) ([]client.Client, error) {
	return m.byHousehold[householdID], nil
}

func (m *mockClientRepo) addClient(c client.Client) {
	m.clients[c.ID] = &c
	m.byAdvisor[c.AdvisorID] = append(m.byAdvisor[c.AdvisorID], c)
	if c.HouseholdID != nil {
		m.byHousehold[*c.HouseholdID] = append(m.byHousehold[*c.HouseholdID], c)
	}
}

// --- Mock HouseholdRepository ---

type mockHouseholdRepo struct {
	households map[string]*client.Household
	byClient   map[string]*client.Household
}

func newMockHouseholdRepo() *mockHouseholdRepo {
	return &mockHouseholdRepo{
		households: make(map[string]*client.Household),
		byClient:   make(map[string]*client.Household),
	}
}

func (m *mockHouseholdRepo) GetHousehold(_ context.Context, id string) (*client.Household, error) {
	h, ok := m.households[id]
	if !ok {
		return nil, errors.New("household not found")
	}
	return h, nil
}

func (m *mockHouseholdRepo) GetHouseholdByClientID(_ context.Context, clientID string) (*client.Household, error) {
	h, ok := m.byClient[clientID]
	if !ok {
		return nil, nil
	}
	return h, nil
}

// --- Mock GoalRepository ---

type mockGoalRepo struct {
	byClient map[string][]client.Goal
}

func newMockGoalRepo() *mockGoalRepo {
	return &mockGoalRepo{byClient: make(map[string][]client.Goal)}
}

func (m *mockGoalRepo) GetGoalsByClientID(_ context.Context, clientID string) ([]client.Goal, error) {
	return m.byClient[clientID], nil
}

// --- Mock AdvisorNoteRepository ---

type mockNoteRepo struct {
	byClient map[string][]client.AdvisorNote
	nextID   int
}

func newMockNoteRepo() *mockNoteRepo {
	return &mockNoteRepo{byClient: make(map[string][]client.AdvisorNote)}
}

func (m *mockNoteRepo) GetNotes(_ context.Context, clientID string, _ string) ([]client.AdvisorNote, error) {
	return m.byClient[clientID], nil
}

func (m *mockNoteRepo) AddNote(_ context.Context, clientID string, advisorID string, text string) (*client.AdvisorNote, error) {
	m.nextID++
	note := client.AdvisorNote{
		ID:        fmt.Sprintf("n%d", m.nextID),
		ClientID:  clientID,
		AdvisorID: advisorID,
		Date:      time.Now(),
		Text:      text,
	}
	m.byClient[clientID] = append(m.byClient[clientID], note)
	return &note, nil
}

// --- Mock AdvisorRepository ---

type mockAdvisorRepo struct {
	advisors map[string]*client.Advisor
}

func newMockAdvisorRepo() *mockAdvisorRepo {
	return &mockAdvisorRepo{advisors: make(map[string]*client.Advisor)}
}

func (m *mockAdvisorRepo) GetAdvisor(_ context.Context, id string) (*client.Advisor, error) {
	a, ok := m.advisors[id]
	if !ok {
		return nil, errors.New("advisor not found")
	}
	return a, nil
}

// --- Mock AccountRepository ---

type mockAccountRepo struct {
	accounts map[string]*account.Account
	byClient map[string][]account.Account
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{
		accounts: make(map[string]*account.Account),
		byClient: make(map[string][]account.Account),
	}
}

func (m *mockAccountRepo) GetAccount(_ context.Context, id string) (*account.Account, error) {
	a, ok := m.accounts[id]
	if !ok {
		return nil, errors.New("account not found")
	}
	return a, nil
}

func (m *mockAccountRepo) GetAccountsByClientID(_ context.Context, clientID string) ([]account.Account, error) {
	return m.byClient[clientID], nil
}

func (m *mockAccountRepo) UpdateFHSALifetimeContributions(_ context.Context, _ string, _ float64) error {
	return nil
}

func (m *mockAccountRepo) addAccount(a account.Account) {
	m.accounts[a.ID] = &a
	m.byClient[a.ClientID] = append(m.byClient[a.ClientID], a)
}

// --- Mock AlertService ---

type mockAlertService struct {
	mu     sync.Mutex
	alerts map[string]*alert.Alert

	sendCalls        []sendCall
	trackCalls       []trackCall
	snoozeCalls      []snoozeCall
	acknowledgeCalls []string
	healthByClient   map[string]alert.HealthStatus
	sendErr          error
	trackErr         error
	snoozeErr        error
	acknowledgeErr   error
}

type sendCall struct {
	AlertID string
	Message *string
}

type trackCall struct {
	AlertID        string
	ActionItemText string
}

type snoozeCall struct {
	AlertID string
	Until   *time.Time
}

func newMockAlertService() *mockAlertService {
	return &mockAlertService{
		alerts:         make(map[string]*alert.Alert),
		healthByClient: make(map[string]alert.HealthStatus),
	}
}

func (m *mockAlertService) ProcessEvent(_ context.Context, _ eventbus.EventEnvelope) error {
	return nil
}

func (m *mockAlertService) Send(_ context.Context, alertID string, message *string) (*alert.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls = append(m.sendCalls, sendCall{alertID, message})
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	a, ok := m.alerts[alertID]
	if !ok {
		return nil, alert.ErrAlertNotFound
	}
	a.Status = alert.StatusActed
	return a, nil
}

func (m *mockAlertService) Track(_ context.Context, alertID string, text string) (*alert.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trackCalls = append(m.trackCalls, trackCall{alertID, text})
	if m.trackErr != nil {
		return nil, m.trackErr
	}
	a, ok := m.alerts[alertID]
	if !ok {
		return nil, alert.ErrAlertNotFound
	}
	a.Status = alert.StatusActed
	return a, nil
}

func (m *mockAlertService) Snooze(_ context.Context, alertID string, until *time.Time) (*alert.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snoozeCalls = append(m.snoozeCalls, snoozeCall{alertID, until})
	if m.snoozeErr != nil {
		return nil, m.snoozeErr
	}
	a, ok := m.alerts[alertID]
	if !ok {
		return nil, alert.ErrAlertNotFound
	}
	a.Status = alert.StatusSnoozed
	return a, nil
}

func (m *mockAlertService) Acknowledge(_ context.Context, alertID string) (*alert.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acknowledgeCalls = append(m.acknowledgeCalls, alertID)
	if m.acknowledgeErr != nil {
		return nil, m.acknowledgeErr
	}
	a, ok := m.alerts[alertID]
	if !ok {
		return nil, alert.ErrAlertNotFound
	}
	a.Status = alert.StatusClosed
	return a, nil
}

func (m *mockAlertService) Close(_ context.Context, alertID string) (*alert.Alert, error) {
	a, ok := m.alerts[alertID]
	if !ok {
		return nil, alert.ErrAlertNotFound
	}
	a.Status = alert.StatusClosed
	return a, nil
}

func (m *mockAlertService) ComputeHealthStatus(_ context.Context, clientID string) (alert.HealthStatus, error) {
	h, ok := m.healthByClient[clientID]
	if !ok {
		return alert.HealthGreen, nil
	}
	return h, nil
}

func (m *mockAlertService) addAlert(a *alert.Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[a.ID] = a
}

// --- Mock AlertRepository ---

type mockAlertRepo struct {
	alerts    map[string]*alert.Alert
	byClient  map[string][]alert.Alert
	byAdvisor map[string][]alert.Alert
}

func newMockAlertRepo() *mockAlertRepo {
	return &mockAlertRepo{
		alerts:    make(map[string]*alert.Alert),
		byClient:  make(map[string][]alert.Alert),
		byAdvisor: make(map[string][]alert.Alert),
	}
}

func (m *mockAlertRepo) FindByConditionKey(_ context.Context, _ string) (*alert.Alert, error) {
	return nil, nil
}

func (m *mockAlertRepo) GetAlert(_ context.Context, id string) (*alert.Alert, error) {
	a, ok := m.alerts[id]
	if !ok {
		return nil, alert.ErrAlertNotFound
	}
	return a, nil
}

func (m *mockAlertRepo) GetAlertsByClientID(_ context.Context, clientID string) ([]alert.Alert, error) {
	return m.byClient[clientID], nil
}

func (m *mockAlertRepo) GetAlertsByAdvisorID(_ context.Context, advisorID string) ([]alert.Alert, error) {
	return m.byAdvisor[advisorID], nil
}

func (m *mockAlertRepo) CreateAlert(_ context.Context, a *alert.Alert) (*alert.Alert, error) {
	m.alerts[a.ID] = a
	return a, nil
}

func (m *mockAlertRepo) UpdateAlert(_ context.Context, a *alert.Alert) (*alert.Alert, error) {
	m.alerts[a.ID] = a
	return a, nil
}

func (m *mockAlertRepo) addAlert(a alert.Alert) {
	m.alerts[a.ID] = &a
	m.byClient[a.ClientID] = append(m.byClient[a.ClientID], a)
}

func (m *mockAlertRepo) addAlertForAdvisor(advisorID string, a alert.Alert) {
	m.alerts[a.ID] = &a
	m.byClient[a.ClientID] = append(m.byClient[a.ClientID], a)
	m.byAdvisor[advisorID] = append(m.byAdvisor[advisorID], a)
}

// --- Mock ActionItemService ---

type mockActionItemService struct {
	items       map[string]*actionitem.ActionItem
	byClient    map[string][]actionitem.ActionItem
	byAlert     map[string][]actionitem.ActionItem
	nextID      int
	updateErr   error
}

func newMockActionItemService() *mockActionItemService {
	return &mockActionItemService{
		items:    make(map[string]*actionitem.ActionItem),
		byClient: make(map[string][]actionitem.ActionItem),
		byAlert:  make(map[string][]actionitem.ActionItem),
	}
}

func (m *mockActionItemService) CreateActionItem(_ context.Context, clientID string, alertID *string, text string, dueDate *time.Time) (*actionitem.ActionItem, error) {
	m.nextID++
	item := &actionitem.ActionItem{
		ID:        fmt.Sprintf("ai%d", m.nextID),
		ClientID:  clientID,
		AlertID:   alertID,
		Text:      text,
		Status:    actionitem.ActionItemStatusPending,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}
	m.items[item.ID] = item
	m.byClient[clientID] = append(m.byClient[clientID], *item)
	if alertID != nil {
		m.byAlert[*alertID] = append(m.byAlert[*alertID], *item)
	}
	return item, nil
}

func (m *mockActionItemService) GetActionItem(_ context.Context, id string) (*actionitem.ActionItem, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, errors.New("action item not found")
	}
	return item, nil
}

func (m *mockActionItemService) GetActionItemsByClientID(_ context.Context, clientID string) ([]actionitem.ActionItem, error) {
	return m.byClient[clientID], nil
}

func (m *mockActionItemService) GetActionItemsByAlertID(_ context.Context, alertID string) ([]actionitem.ActionItem, error) {
	return m.byAlert[alertID], nil
}

func (m *mockActionItemService) UpdateActionItem(_ context.Context, id string, text *string, status *actionitem.ActionItemStatus, dueDate *time.Time) (*actionitem.ActionItem, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	item, ok := m.items[id]
	if !ok {
		return nil, errors.New("action item not found")
	}
	if text != nil {
		item.Text = *text
	}
	if status != nil {
		item.Status = *status
	}
	if dueDate != nil {
		item.DueDate = dueDate
	}
	return item, nil
}

func (m *mockActionItemService) CloseActionItem(_ context.Context, id string, note string) (*actionitem.ActionItem, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, errors.New("action item not found")
	}
	item.Status = actionitem.ActionItemStatusClosed
	item.ResolutionNote = &note
	return item, nil
}

func (m *mockActionItemService) addItem(item actionitem.ActionItem) {
	m.items[item.ID] = &item
	m.byClient[item.ClientID] = append(m.byClient[item.ClientID], item)
	if item.AlertID != nil {
		m.byAlert[*item.AlertID] = append(m.byAlert[*item.AlertID], item)
	}
}

// --- Mock ContributionEngine ---

type mockContribEngine struct {
	summaries      map[string]*contribution.ContributionSummary
	analyzeClients []string
	analyzeErr     map[string]error
}

func newMockContribEngine() *mockContribEngine {
	return &mockContribEngine{
		summaries:  make(map[string]*contribution.ContributionSummary),
		analyzeErr: make(map[string]error),
	}
}

func (m *mockContribEngine) AnalyzeClient(_ context.Context, clientID string, _ int) error {
	m.analyzeClients = append(m.analyzeClients, clientID)
	if err, ok := m.analyzeErr[clientID]; ok {
		return err
	}
	return nil
}

func (m *mockContribEngine) GetContributionSummary(_ context.Context, clientID string, taxYear int) (*contribution.ContributionSummary, error) {
	key := fmt.Sprintf("%s:%d", clientID, taxYear)
	s, ok := m.summaries[key]
	if !ok {
		return nil, errors.New("contribution summary not found")
	}
	return s, nil
}

func (m *mockContribEngine) GetRoom(_ context.Context, _ string, _ string, _ int) (float64, error) {
	return 0, nil
}

func (m *mockContribEngine) RecordContribution(_ context.Context, c *contribution.Contribution) (*contribution.Contribution, error) {
	return c, nil
}

// --- Mock TransferRepository ---

type mockTransferRepo struct {
	transfers map[string]*transfer.Transfer
	byClient  map[string][]transfer.Transfer
}

func newMockTransferRepo() *mockTransferRepo {
	return &mockTransferRepo{
		transfers: make(map[string]*transfer.Transfer),
		byClient:  make(map[string][]transfer.Transfer),
	}
}

func (m *mockTransferRepo) GetTransfer(_ context.Context, id string) (*transfer.Transfer, error) {
	t, ok := m.transfers[id]
	if !ok {
		return nil, errors.New("transfer not found")
	}
	return t, nil
}

func (m *mockTransferRepo) GetTransfersByClientID(_ context.Context, clientID string) ([]transfer.Transfer, error) {
	return m.byClient[clientID], nil
}

func (m *mockTransferRepo) GetActiveTransfers(_ context.Context) ([]transfer.Transfer, error) {
	return nil, nil
}

func (m *mockTransferRepo) CreateTransfer(_ context.Context, t *transfer.Transfer) (*transfer.Transfer, error) {
	return t, nil
}

func (m *mockTransferRepo) UpdateTransferStatus(_ context.Context, _ string, _ transfer.TransferStatus) (*transfer.Transfer, error) {
	return nil, nil
}

func (m *mockTransferRepo) addTransfer(t transfer.Transfer) {
	m.transfers[t.ID] = &t
	m.byClient[t.ClientID] = append(m.byClient[t.ClientID], t)
}

// --- Mock TransferMonitor ---

type mockTransferMonitor struct {
	called bool
	err    error
}

func newMockTransferMonitor() *mockTransferMonitor {
	return &mockTransferMonitor{}
}

func (m *mockTransferMonitor) CheckStuckTransfers(_ context.Context) ([]transfer.TransferCheckResult, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}

// --- Mock TemporalScanner ---

type mockTemporalScanner struct {
	called bool
	err    error
}

func newMockTemporalScanner() *mockTemporalScanner {
	return &mockTemporalScanner{}
}

func (m *mockTemporalScanner) RunSweep(_ context.Context, _ string, _ time.Time) (*temporal.ScannerResult, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return &temporal.ScannerResult{}, nil
}

// --- Helper to build a Resolver with mocks ---

type testFixture struct {
	resolver       *Resolver
	clientRepo     *mockClientRepo
	householdRepo  *mockHouseholdRepo
	goalRepo       *mockGoalRepo
	noteRepo       *mockNoteRepo
	advisorRepo    *mockAdvisorRepo
	accountRepo    *mockAccountRepo
	alertService   *mockAlertService
	alertRepo      *mockAlertRepo
	actionItemSvc  *mockActionItemService
	contribEngine  *mockContribEngine
	transferRepo   *mockTransferRepo
	transferMon    *mockTransferMonitor
	temporalScan   *mockTemporalScanner
	eventBus       eventbus.EventBus
}

func newTestFixture() *testFixture {
	cr := newMockClientRepo()
	hr := newMockHouseholdRepo()
	gr := newMockGoalRepo()
	nr := newMockNoteRepo()
	ar := newMockAdvisorRepo()
	acr := newMockAccountRepo()
	as := newMockAlertService()
	alr := newMockAlertRepo()
	ais := newMockActionItemService()
	ce := newMockContribEngine()
	tr := newMockTransferRepo()
	tm := newMockTransferMonitor()
	ts := newMockTemporalScanner()
	bus := eventbus.New()

	return &testFixture{
		resolver: &Resolver{
			ClientRepo:        cr,
			HouseholdRepo:     hr,
			GoalRepo:          gr,
			NoteRepo:          nr,
			AdvisorRepo:       ar,
			AccountRepo:       acr,
			AlertService:      as,
			AlertRepo:         alr,
			ActionItemService: ais,
			ContribEngine:     ce,
			TransferRepo:      tr,
			TransferMonitor:   tm,
			TemporalScanner:   ts,
			EventBus:          bus,
		},
		clientRepo:    cr,
		householdRepo: hr,
		goalRepo:      gr,
		noteRepo:      nr,
		advisorRepo:   ar,
		accountRepo:   acr,
		alertService:  as,
		alertRepo:     alr,
		actionItemSvc: ais,
		contribEngine: ce,
		transferRepo:  tr,
		transferMon:   tm,
		temporalScan:  ts,
		eventBus:      bus,
	}
}
