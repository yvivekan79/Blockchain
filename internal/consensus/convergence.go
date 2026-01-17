package consensus

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ProtocolConvergenceManager manages convergence across different consensus algorithms
type ProtocolConvergenceManager struct {
	protocols      map[string]Consensus
	activeProtocol string
	convergenceLog map[string]*ConvergenceState
	mu             sync.RWMutex
	logger         interface{ LogConsensus(algorithm, action string, fields logrus.Fields) }
}

// ConvergenceState tracks convergence status for each protocol
type ConvergenceState struct {
	LastBlockHeight int64
	LastBlockTime   time.Time
	SuccessRate     float64
	ErrorCount      int64
	ViewChanges     int64
	Status          string // "converged", "diverging", "failed"
}

// NewProtocolConvergenceManager creates a new convergence manager
func NewProtocolConvergenceManager(logger interface{ LogConsensus(algorithm, action string, fields logrus.Fields) }) *ProtocolConvergenceManager {
	return &ProtocolConvergenceManager{
		protocols:      make(map[string]Consensus),
		convergenceLog: make(map[string]*ConvergenceState),
		logger:         logger,
	}
}

// RegisterProtocol registers a consensus protocol
func (pcm *ProtocolConvergenceManager) RegisterProtocol(name string, protocol Consensus) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	pcm.protocols[name] = protocol
	pcm.convergenceLog[name] = &ConvergenceState{
		Status:      "initialized",
		SuccessRate: 0.0,
	}

	pcm.logger.LogConsensus("convergence", "protocol_registered", logrus.Fields{
		"protocol": name,
		"timestamp": time.Now().UTC(),
	})
}

// SetActiveProtocol sets the currently active protocol
func (pcm *ProtocolConvergenceManager) SetActiveProtocol(name string) error {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	if _, exists := pcm.protocols[name]; !exists {
		return fmt.Errorf("protocol %s not registered", name)
	}

	pcm.activeProtocol = name

	pcm.logger.LogConsensus("convergence", "protocol_activated", logrus.Fields{
		"protocol": name,
		"timestamp": time.Now().UTC(),
	})

	return nil
}

// UpdateConvergenceStatus updates convergence status for a protocol
func (pcm *ProtocolConvergenceManager) UpdateConvergenceStatus(protocol string, blockHeight int64, success bool) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	state, exists := pcm.convergenceLog[protocol]
	if !exists {
		state = &ConvergenceState{}
		pcm.convergenceLog[protocol] = state
	}

	state.LastBlockHeight = blockHeight
	state.LastBlockTime = time.Now()

	if success {
		state.SuccessRate = (state.SuccessRate*0.9) + (1.0*0.1) // Moving average
		if state.SuccessRate > 0.8 {
			state.Status = "converged"
		}
	} else {
		state.ErrorCount++
		state.SuccessRate = state.SuccessRate * 0.9 // Decay on failure
		if state.SuccessRate < 0.3 {
			state.Status = "diverging"
		}
	}

	pcm.logger.LogConsensus("convergence", "status_updated", logrus.Fields{
		"protocol":      protocol,
		"block_height":  blockHeight,
		"success":       success,
		"success_rate":  state.SuccessRate,
		"status":        state.Status,
		"error_count":   state.ErrorCount,
		"timestamp":     time.Now().UTC(),
	})
}

// GetConvergenceReport returns convergence status for all protocols
func (pcm *ProtocolConvergenceManager) GetConvergenceReport() map[string]*ConvergenceState {
	pcm.mu.RLock()
	defer pcm.mu.RUnlock()

	report := make(map[string]*ConvergenceState)
	for name, state := range pcm.convergenceLog {
		// Create a copy to avoid race conditions
		report[name] = &ConvergenceState{
			LastBlockHeight: state.LastBlockHeight,
			LastBlockTime:   state.LastBlockTime,
			SuccessRate:     state.SuccessRate,
			ErrorCount:      state.ErrorCount,
			ViewChanges:     state.ViewChanges,
			Status:          state.Status,
		}
	}

	return report
}

// IsConverged checks if all registered protocols are converging
func (pcm *ProtocolConvergenceManager) IsConverged() bool {
	pcm.mu.RLock()
	defer pcm.mu.RUnlock()

	for _, state := range pcm.convergenceLog {
		if state.Status != "converged" && state.Status != "initialized" {
			return false
		}
	}

	return true
}

// LogViewChange logs a view change for convergence tracking
func (pcm *ProtocolConvergenceManager) LogViewChange(protocol string) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	state, exists := pcm.convergenceLog[protocol]
	if !exists {
		state = &ConvergenceState{}
		pcm.convergenceLog[protocol] = state
	}

	state.ViewChanges++

	// Too many view changes indicate divergence
	if state.ViewChanges > 10 {
		state.Status = "diverging"
	}

	pcm.logger.LogConsensus("convergence", "view_change_logged", logrus.Fields{
		"protocol":     protocol,
		"view_changes": state.ViewChanges,
		"status":       state.Status,
		"timestamp":    time.Now().UTC(),
	})
}