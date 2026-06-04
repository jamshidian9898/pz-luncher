package launchstate

import (
	"errors"
	"sync"

	"pzlauncher/libs/contracts"
)

type simpleMachine struct {
	mu    sync.Mutex
	state contracts.LaunchState
}

func NewSimpleMachine() StateMachine {
	return &simpleMachine{state: contracts.LaunchStateIdle}
}

func (s *simpleMachine) CurrentState() contracts.LaunchState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *simpleMachine) Transition(next contracts.LaunchState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// simple validation: allow all transitions except to empty
	if next == "" {
		return errors.New("invalid next state")
	}
	s.state = next
	return nil
}
