package service

import (
	"sync"
)

// ObjectStore — потокобезопасное хранилище маппинга номер объявления → объект
type ObjectStore struct {
	mu      sync.RWMutex
	objects map[string]string // listingNumber → objectName
}

func NewObjectStore() *ObjectStore {
	return &ObjectStore{objects: make(map[string]string)}
}

func (s *ObjectStore) Set(listingNumber, objectName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[listingNumber] = objectName
}

func (s *ObjectStore) Get(listingNumber string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.objects[listingNumber]
	return obj, ok
}

func (s *ObjectStore) Remove(listingNumber string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, listingNumber)
}

func (s *ObjectStore) List() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string, len(s.objects))
	for k, v := range s.objects {
		result[k] = v
	}
	return result
}

func (s *ObjectStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.objects)
}

// LoadFromRows загружает маппинг из строк [listingNumber, objectName]
func (s *ObjectStore) LoadFromRows(rows [][]string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		num, obj := row[0], row[1]
		if num != "" && obj != "" {
			s.objects[num] = obj
			count++
		}
	}
	return count
}
