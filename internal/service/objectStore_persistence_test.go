package service

import "testing"

type fakeObjectMappingRepository struct {
	mappings map[string]string
	onUpsert func()
}

func (r *fakeObjectMappingRepository) LoadAll() (map[string]string, error) {
	result := make(map[string]string, len(r.mappings))
	for number, object := range r.mappings {
		result[number] = object
	}
	return result, nil
}

func (r *fakeObjectMappingRepository) Upsert(mappings map[string]string) error {
	if r.onUpsert != nil {
		r.onUpsert()
	}
	for number, object := range mappings {
		r.mappings[number] = object
	}
	return nil
}

func TestObjectStoreSerializesPersistenceAndCacheMutation(t *testing.T) {
	repository := &fakeObjectMappingRepository{mappings: make(map[string]string)}
	store, err := newPersistentObjectStore(repository)
	if err != nil {
		t.Fatalf("создание хранилища: %v", err)
	}

	persistenceCalledWithoutStoreLock := false
	repository.onUpsert = func() {
		if store.mu.TryLock() {
			persistenceCalledWithoutStoreLock = true
			store.mu.Unlock()
		}
	}
	if _, err := store.LoadFromRows([][]string{{"1001", "Объект"}}); err != nil {
		t.Fatalf("incremental upload: %v", err)
	}
	if persistenceCalledWithoutStoreLock {
		t.Fatal("запись в БД и обновление кэша должны выполняться под одной блокировкой")
	}
}

func (r *fakeObjectMappingRepository) Delete(listingNumber string) error {
	delete(r.mappings, listingNumber)
	return nil
}

func TestObjectStoreIncrementalUploadPersistsAndKeepsOldMappings(t *testing.T) {
	repository := &fakeObjectMappingRepository{mappings: map[string]string{
		"1001": "Старое название",
		"1003": "Сохранить без изменений",
	}}
	store, err := newPersistentObjectStore(repository)
	if err != nil {
		t.Fatalf("создание хранилища: %v", err)
	}

	count, err := store.LoadFromRows([][]string{
		{"1001", "Новое название"},
		{"1002", "Новый объект"},
	})
	if err != nil {
		t.Fatalf("incremental upload: %v", err)
	}
	if count != 2 {
		t.Fatalf("ожидалось 2 добавленных/обновлённых строки, получено %d", count)
	}

	restartedStore, err := newPersistentObjectStore(repository)
	if err != nil {
		t.Fatalf("повторное создание хранилища: %v", err)
	}
	want := map[string]string{
		"1001": "Новое название",
		"1002": "Новый объект",
		"1003": "Сохранить без изменений",
	}
	got := restartedStore.List()
	if len(got) != len(want) {
		t.Fatalf("после перезапуска ожидалось %d связей, получено %d: %#v", len(want), len(got), got)
	}
	for number, object := range want {
		if got[number] != object {
			t.Errorf("маппинг %s: ожидался %q, получен %q", number, object, got[number])
		}
	}
}
