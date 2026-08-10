package fsm

import "time"

func (f *FSM[A, T]) GetLastState() T {
	return f.lastState
}

func (f *FSM[A, T]) GetCurrentState() T {
	return f.currentState
}

func (f *FSM[A, T]) GetStateLeftDuration() time.Duration {
	return f.stateLeftDuration
}

func (f *FSM[A, T]) SetStateLeftDuration(value time.Duration) {
	f.stateLeftDuration = value
}

func (f *FSM[A, T]) GetStateTotalDuration() time.Duration {
	return f.stateTotalDuration
}

func (f *FSM[A, T]) GetExtraSpeed100() int64 {
	return f.extraSpeed100
}

func (f *FSM[A, T]) SetExtraSpeed100(value int64) {
	f.extraSpeed100 = value
}
