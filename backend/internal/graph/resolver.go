package graph

import (
	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/contribution"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
	"github.com/swatkatz/advisorhub/backend/internal/temporal"
	"github.com/swatkatz/advisorhub/backend/internal/transfer"
)

// Resolver is the root resolver — dependency injection container.
type Resolver struct {
	ClientRepo        client.ClientRepository
	HouseholdRepo     client.HouseholdRepository
	GoalRepo          client.GoalRepository
	NoteRepo          client.AdvisorNoteRepository
	AdvisorRepo       client.AdvisorRepository
	AccountRepo       account.AccountRepository
	AlertService      alert.AlertService
	AlertRepo         alert.AlertRepository
	ActionItemService actionitem.ActionItemService
	ContribEngine     contribution.ContributionEngine
	TransferRepo      transfer.TransferRepository
	TransferMonitor   transfer.TransferMonitor
	TemporalScanner   temporal.TemporalScanner
	EventBus          eventbus.EventBus
}
