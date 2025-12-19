package bot

import "sync"

// NavigationStack tracks each user's menu navigation history.
// This allows the back button to work correctly regardless of menu depth.
type NavigationStack struct {
	mu     sync.RWMutex
	stacks map[int64][]MenuType // userID -> stack of visited menus
}

// NewNavigationStack creates a new navigation stack manager.
func NewNavigationStack() *NavigationStack {
	return &NavigationStack{
		stacks: make(map[int64][]MenuType),
	}
}

// Push adds a menu to the user's navigation history.
func (ns *NavigationStack) Push(userID int64, menu MenuType) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.stacks[userID] == nil {
		ns.stacks[userID] = make([]MenuType, 0, 5)
	}

	ns.stacks[userID] = append(ns.stacks[userID], menu)
}

// Pop removes the last menu from user's navigation history.
func (ns *NavigationStack) Pop(userID int64) MenuType {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	stack := ns.stacks[userID]
	if len(stack) == 0 {
		return MenuMain
	}

	last := stack[len(stack)-1]
	ns.stacks[userID] = stack[:len(stack)-1]
	return last
}

// Current returns the current menu without removing it.
func (ns *NavigationStack) Current(userID int64) MenuType {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	stack := ns.stacks[userID]
	if len(stack) == 0 {
		return MenuMain
	}

	return stack[len(stack)-1]
}

// Reset clears the navigation history for a user.
func (ns *NavigationStack) Reset(userID int64) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	delete(ns.stacks, userID)
}

// Depth returns how deep the user is in the menu tree.
func (ns *NavigationStack) Depth(userID int64) int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return len(ns.stacks[userID])
}
