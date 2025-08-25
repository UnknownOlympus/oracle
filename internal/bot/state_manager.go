package bot

import "sync"

// UserState saves a context for next message from user.
type UserState struct {
	WaitingFor string
	TaskID     int
}

// StateManager manages the states of all users.
type StateManager struct {
	mu     sync.Mutex
	states map[int64]UserState
}

func NewStateManager() *StateManager {
	return &StateManager{states: make(map[int64]UserState)}
}

// Set sets the state for the user.
func (sm *StateManager) Set(userID int64, state UserState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.states[userID] = state
}

// Get gets and immediately delete user state.
func (sm *StateManager) Get(userID int64) (UserState, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, ok := sm.states[userID]
	if ok {
		delete(sm.states, userID)
	}
	return state, ok
}
