package launchstate

import "pzlauncher/libs/contracts"

type StateMachine interface {
	CurrentState() contracts.LaunchState
	Transition(next contracts.LaunchState) error
}
