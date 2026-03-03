package transfer

import (
	"context"
	"fmt"
	"math"
	"time"
)

// TransferStatus represents the pipeline stage of a transfer.
type TransferStatus string

const (
	StatusInitiated          TransferStatus = "INITIATED"
	StatusDocumentsSubmitted TransferStatus = "DOCUMENTS_SUBMITTED"
	StatusInReview           TransferStatus = "IN_REVIEW"
	StatusInTransit          TransferStatus = "IN_TRANSIT"
	StatusReceived           TransferStatus = "RECEIVED"
	StatusInvested           TransferStatus = "INVESTED"
)

// statusOrder defines the forward-only pipeline for validation.
var statusOrder = []TransferStatus{
	StatusInitiated,
	StatusDocumentsSubmitted,
	StatusInReview,
	StatusInTransit,
	StatusReceived,
	StatusInvested,
}

// nextStatus returns the only valid next status, or an error if terminal.
func nextStatus(current TransferStatus) (TransferStatus, error) {
	for i, s := range statusOrder {
		if s == current {
			if i+1 < len(statusOrder) {
				return statusOrder[i+1], nil
			}
			return "", fmt.Errorf("status %s is terminal: no valid transition", current)
		}
	}
	return "", fmt.Errorf("unknown status: %s", current)
}

// StageThreshold defines how many days a transfer can sit in a stage before it's considered stuck.
var StageThreshold = map[TransferStatus]int{
	StatusInitiated:          5,
	StatusDocumentsSubmitted: 10,
	StatusInReview:           14,
	StatusInTransit:          14,
	StatusReceived:           5,
}

// CheckSignal indicates the outcome of checking a single transfer for stuck status.
type CheckSignal string

const (
	SignalStuckDetected CheckSignal = "STUCK_DETECTED"
	SignalNoChange      CheckSignal = "NO_CHANGE"
)

// Account type constants (denormalized on Transfer for display).
const (
	AccountTypeRRSP   = "RRSP"
	AccountTypeTFSA   = "TFSA"
	AccountTypeFHSA   = "FHSA"
	AccountTypeRESP   = "RESP"
	AccountTypeNonReg = "NON_REG"
)

// Event type constants.
const (
	EventTransferStuck         = "TransferStuck"
	EventTransferCompleted     = "TransferCompleted"
	EventTransferStatusChanged = "TransferStatusChanged"
)

// Transfer represents a money transfer between institutions.
type Transfer struct {
	ID                string
	ClientID          string
	SourceInstitution string
	AccountType       string
	Amount            float64
	Status            TransferStatus
	InitiatedAt       time.Time
	LastStatusChange  time.Time
}

// DaysInCurrentStage computes how many full days the transfer has been in its current stage.
func (t *Transfer) DaysInCurrentStage(now time.Time) int {
	return int(math.Floor(now.Sub(t.LastStatusChange).Hours() / 24))
}

// IsStuck returns true if the transfer has exceeded its stage threshold.
// INVESTED transfers are never stuck.
func (t *Transfer) IsStuck(now time.Time) bool {
	if t.Status == StatusInvested {
		return false
	}
	threshold, ok := StageThreshold[t.Status]
	if !ok {
		return false
	}
	return t.DaysInCurrentStage(now) > threshold
}

// TransferCheckResult is returned by CheckStuckTransfers for each active transfer.
type TransferCheckResult struct {
	TransferID string
	Signal     CheckSignal
}

// TransferRepository defines data access for transfers.
type TransferRepository interface {
	GetTransfer(ctx context.Context, id string) (*Transfer, error)
	GetTransfersByClientID(ctx context.Context, clientID string) ([]Transfer, error)
	GetActiveTransfers(ctx context.Context) ([]Transfer, error)
	CreateTransfer(ctx context.Context, transfer *Transfer) (*Transfer, error)
	UpdateTransferStatus(ctx context.Context, id string, newStatus TransferStatus) (*Transfer, error)
}

// TransferMonitor defines the stuck detection interface.
type TransferMonitor interface {
	CheckStuckTransfers(ctx context.Context) ([]TransferCheckResult, error)
}
