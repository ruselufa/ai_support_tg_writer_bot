package bot

import (
	"sync"
)

// UserState представляет состояние пользователя
type UserState struct {
	IsReplyingToTicket bool
	TicketID           uint
}

// StateManager управляет состояниями пользователей
type StateManager struct {
	states map[int64]*UserState
	mutex  sync.RWMutex
}

// NewStateManager создает новый менеджер состояний
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[int64]*UserState),
	}
}

// SetUserState устанавливает состояние пользователя
func (sm *StateManager) SetUserState(userID int64, state *UserState) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.states[userID] = state
}

// GetUserState получает состояние пользователя
func (sm *StateManager) GetUserState(userID int64) *UserState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.states[userID]
}

// ClearUserState очищает состояние пользователя
func (sm *StateManager) ClearUserState(userID int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.states, userID)
}

// SetReplyingState устанавливает состояние ответа на тикет
func (sm *StateManager) SetReplyingState(userID int64, ticketID uint) {
	sm.SetUserState(userID, &UserState{
		IsReplyingToTicket: true,
		TicketID:           ticketID,
	})
}

// IsReplyingToTicket проверяет, отвечает ли пользователь на тикет
func (sm *StateManager) IsReplyingToTicket(userID int64) (bool, uint) {
	state := sm.GetUserState(userID)
	if state != nil && state.IsReplyingToTicket {
		return true, state.TicketID
	}
	return false, 0
}

