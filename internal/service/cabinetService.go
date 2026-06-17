package service

import (
	"sync"
	"github.com/google/uuid"
)

type Cabinet struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	UserID uint   `json:"userId"`
}

type CabinetStore struct {
	mu       sync.RWMutex
	cabinets map[string]*Cabinet
}

func NewCabinetStore() *CabinetStore {
	return &CabinetStore{cabinets: make(map[string]*Cabinet)}
}

func (s *CabinetStore) Create(name string, userID uint) *Cabinet {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &Cabinet{ID: uuid.New().String(), Name: name, UserID: userID}
	s.cabinets[c.ID] = c
	return c
}

func (s *CabinetStore) Get(id string) *Cabinet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cabinets[id]
}

func (s *CabinetStore) ListByUser(userID uint) []Cabinet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Cabinet, 0)
	for _, c := range s.cabinets {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result
}

func (s *CabinetStore) Delete(id string, userID uint) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cabinets[id]
	if !ok || c.UserID != userID {
		return false
	}
	delete(s.cabinets, id)
	return true
}
