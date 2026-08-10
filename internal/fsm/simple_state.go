package fsm

import "time"

type SimpleState[A any, T comparable] struct {
	MPreEnter              func(lastState T, arg A, isTimeOut bool) time.Duration
	MPostEnter             func(lastState T, arg A, isTimeOut bool)
	MUpdate                func(arg A, dt time.Duration)
	MPreExist              func(arg A, nextState T, isTimeOut bool)
	MPostExist             func(arg A, nextState T, isTimeOut bool)
	MGetNextStateOnTimeOut func(arg A) T
	MOnEventTrigger        func(arg A, event any)
}

func (s *SimpleState[A, T]) PreEnter(lastState T, arg A, isTimeOut bool) time.Duration {
	if s.MPreEnter != nil {
		return s.MPreEnter(lastState, arg, isTimeOut)
	}
	return -1
}

func (s *SimpleState[A, T]) PostEnter(lastState T, arg A, isTimeOut bool) {
	if s.MPostEnter != nil {
		s.MPostEnter(lastState, arg, isTimeOut)
	}
}

func (s *SimpleState[A, T]) Update(arg A, dt time.Duration) {
	if s.MUpdate != nil {
		s.MUpdate(arg, dt)
	}
}

func (s *SimpleState[A, T]) PreExist(arg A, nextState T, isTimeOut bool) {
	if s.MPreExist != nil {
		s.MPreExist(arg, nextState, isTimeOut)
	}
}

func (s *SimpleState[A, T]) PostExist(arg A, nextState T, isTimeOut bool) {
	if s.MPostExist != nil {
		s.MPostExist(arg, nextState, isTimeOut)
	}
}

func (s *SimpleState[A, T]) GetNextStateOnTimeOut(arg A) (state T) {
	if s.MGetNextStateOnTimeOut != nil {
		return s.MGetNextStateOnTimeOut(arg)
	}
	return state
}

func (s *SimpleState[A, T]) OnEventTrigger(arg A, event any) {
	if s.MOnEventTrigger != nil {
		s.MOnEventTrigger(arg, event)
	}
}
