package fsm

import "time"

type IState[A any, T comparable] interface {
	PreEnter(lastState T, arg A, isTimeOut bool) time.Duration
	PostEnter(lastState T, arg A, isTimeOut bool)
	Update(arg A, dt time.Duration)
	PreExist(arg A, nextState T, isTimeOut bool)
	PostExist(arg A, nextState T, isTimeOut bool)
	GetNextStateOnTimeOut(arg A) T
	OnEventTrigger(arg A, event any)
}

type SwitchStateFunc[A any, T comparable] func(oldState T, newState T, arg A)

type registry[A any, T comparable] struct {
	defaultState    T
	states          map[T]IState[A, T]
	preSwitchState  SwitchStateFunc[A, T]
	postSwitchState SwitchStateFunc[A, T]
}

type FSM[A any, T comparable] struct {
	registry[A, T]
	lastState          T
	currentState       T
	stateLeftDuration  time.Duration
	stateTotalDuration time.Duration
	extraSpeed100      int64
	curStateInfo       IState[A, T]
}

func NewFsm[A any, T comparable](defaultState T) *FSM[A, T] {
	f := &FSM[A, T]{}
	f.Init(defaultState)
	return f
}

func (f *FSM[A, T]) Init(defaultState T) {
	f.registry = registry[A, T]{
		defaultState: defaultState,
		states:       make(map[T]IState[A, T]),
	}
	f.currentState = defaultState
	f.lastState = defaultState
}

func (f *FSM[A, T]) Recover(initF func()) {
	f.recover(f.defaultState, initF)
}

func (f *FSM[A, T]) RecoverWithDefault(defaultState T, initF func()) {
	f.recover(defaultState, initF)
}

func (f *FSM[A, T]) recover(defaultState T, initF func()) {
	currentState := f.GetCurrentState()
	f.registry = registry[A, T]{
		defaultState: defaultState,
		states:       make(map[T]IState[A, T]),
	}
	if initF != nil {
		initF()
	}
	f.curStateInfo = f.states[currentState]
}

func (f *FSM[A, T]) Register(state T, value IState[A, T]) {
	if f.states == nil {
		f.states = make(map[T]IState[A, T])
	}
	f.states[state] = value
	if state == f.GetCurrentState() {
		f.curStateInfo = value
	}
}

func (f *FSM[A, T]) Switch(arg A, newState T) {
	f.switchState(arg, newState, false)
}

func (f *FSM[A, T]) switchState(arg A, newState T, isTimeOut bool) {
	if f.curStateInfo == nil {
		f.curStateInfo = f.states[f.GetCurrentState()]
	}

	oldState := f.GetCurrentState()
	oldStateInfo := f.curStateInfo
	newStateInfo := f.states[newState]
	if oldStateInfo == nil || newStateInfo == nil {
		return
	}

	if f.preSwitchState != nil {
		f.preSwitchState(oldState, newState, arg)
	}
	oldStateInfo.PreExist(arg, newState, isTimeOut)
	stateDuration := newStateInfo.PreEnter(oldState, arg, isTimeOut)

	f.currentState = newState
	f.curStateInfo = newStateInfo
	f.lastState = oldState
	f.SetStateLeftDuration(stateDuration)
	f.stateTotalDuration = stateDuration

	oldStateInfo.PostExist(arg, newState, isTimeOut)
	newStateInfo.PostEnter(oldState, arg, isTimeOut)
	if f.postSwitchState != nil {
		f.postSwitchState(oldState, newState, arg)
	}
}

func (f *FSM[A, T]) Update(arg A, dt time.Duration) {
	if f.curStateInfo == nil {
		f.currentState = f.defaultState
		f.curStateInfo = f.states[f.defaultState]
	}
	if f.curStateInfo == nil {
		return
	}

	scale := (100.0 + float64(f.GetExtraSpeed100())) / 100.0
	realDT := time.Duration(float64(dt) * scale)
	if f.GetStateTotalDuration() >= 0 {
		f.SetStateLeftDuration(f.GetStateLeftDuration() - realDT)
		if f.GetStateLeftDuration() < 0 {
			nextState := f.curStateInfo.GetNextStateOnTimeOut(arg)
			f.switchState(arg, nextState, true)
		}
	}
	f.curStateInfo.Update(arg, realDT)
}

func (f *FSM[A, T]) TriggerEvent(arg A, event any) {
	if f.curStateInfo == nil {
		f.curStateInfo = f.states[f.GetCurrentState()]
	}
	if f.curStateInfo != nil {
		f.curStateInfo.OnEventTrigger(arg, event)
	}
}

func (f *FSM[A, T]) PreSwitchState(preF SwitchStateFunc[A, T]) {
	f.preSwitchState = preF
}

func (f *FSM[A, T]) PostSwitchState(postF SwitchStateFunc[A, T]) {
	f.postSwitchState = postF
}

func (f *FSM[A, T]) LastState() T {
	return f.GetLastState()
}

func (f *FSM[A, T]) CurrentState() T {
	return f.GetCurrentState()
}

func (f *FSM[A, T]) CurrentStateLeftDuration() time.Duration {
	return f.GetStateLeftDuration()
}

func (f *FSM[A, T]) SetSpeed100(speed100 int64) {
	f.SetExtraSpeed100(speed100)
}

func (f *FSM[A, T]) GetState(state T) IState[A, T] {
	return f.states[state]
}
