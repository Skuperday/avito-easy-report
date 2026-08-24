package service

import (
	"avito-easy-report/internal/database"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type objectMappingRepository interface {
	LoadAll() (map[string]string, error)
	Upsert(map[string]string) error
	Delete(listingNumber string) error
}

type gormObjectMappingRepository struct {
	db *gorm.DB
}

func (r *gormObjectMappingRepository) LoadAll() (map[string]string, error) {
	var rows []database.ObjectMapping
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.ListingNumber] = row.ObjectName
	}
	return result, nil
}

func (r *gormObjectMappingRepository) Upsert(mappings map[string]string) error {
	rows := make([]database.ObjectMapping, 0, len(mappings))
	for listingNumber, objectName := range mappings {
		rows = append(rows, database.ObjectMapping{
			ListingNumber: listingNumber,
			ObjectName:    objectName,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "listing_number"}},
		DoUpdates: clause.AssignmentColumns([]string{"object_name", "updated_at"}),
	}).Create(&rows).Error
}

func (r *gormObjectMappingRepository) Delete(listingNumber string) error {
	return r.db.Delete(&database.ObjectMapping{}, "listing_number = ?", listingNumber).Error
}

// ObjectStore — потокобезопасный кэш постоянного маппинга номер объявления → объект.
type ObjectStore struct {
	mu         sync.RWMutex
	objects    map[string]string
	repository objectMappingRepository
}

// NewObjectStore создаёт временное in-memory хранилище (используется в тестах и fallback-сценариях).
func NewObjectStore() *ObjectStore {
	return &ObjectStore{objects: make(map[string]string)}
}

// NewPersistentObjectStore загружает существующий маппинг из PostgreSQL.
func NewPersistentObjectStore(db *gorm.DB) (*ObjectStore, error) {
	return newPersistentObjectStore(&gormObjectMappingRepository{db: db})
}

func newPersistentObjectStore(repository objectMappingRepository) (*ObjectStore, error) {
	objects, err := repository.LoadAll()
	if err != nil {
		return nil, err
	}
	return &ObjectStore{objects: objects, repository: repository}, nil
}

func (s *ObjectStore) Set(listingNumber, objectName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mappings := map[string]string{listingNumber: objectName}
	if s.repository != nil {
		if err := s.repository.Upsert(mappings); err != nil {
			return err
		}
	}
	s.objects[listingNumber] = objectName
	return nil
}

func (s *ObjectStore) Get(listingNumber string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.objects[listingNumber]
	return obj, ok
}

func (s *ObjectStore) Remove(listingNumber string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repository != nil {
		if err := s.repository.Delete(listingNumber); err != nil {
			return err
		}
	}
	delete(s.objects, listingNumber)
	return nil
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

// LoadFromRows добавляет новые связи и обновляет существующие, не удаляя остальные.
func (s *ObjectStore) LoadFromRows(rows [][]string) (int, error) {
	mappings := make(map[string]string)
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		num, obj := row[0], row[1]
		if num != "" && obj != "" {
			mappings[num] = obj
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repository != nil {
		if err := s.repository.Upsert(mappings); err != nil {
			return 0, err
		}
	}
	for num, obj := range mappings {
		s.objects[num] = obj
	}
	return len(mappings), nil
}
