package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreOperations(t *testing.T) {
	// Создаём временную директорию
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Тест создания нового состояния
	if err := store.Load(); err != nil {
		t.Fatalf("ошибка загрузки: %v", err)
	}

	// Тест добавления ресурса
	r := NewResource("test-1", "test-vm", ResourceDomain)
	r.Status = StatusRunning
	r.SetLibvirtID("uuid-1234")
	r.SetIP("10.0.0.10")
	store.AddResource(r)

	// Тест поиска по ID
	found := store.GetResource("test-1")
	if found == nil {
		t.Fatal("ресурс не найден по ID")
	}
	if found.Name != "test-vm" {
		t.Errorf("ожидалось имя 'test-vm', получено '%s'", found.Name)
	}
	if found.LibvirtID != "uuid-1234" {
		t.Errorf("ожидался libvirt_id 'uuid-1234', получено '%s'", found.LibvirtID)
	}
	if found.IP != "10.0.0.10" {
		t.Errorf("ожидался IP '10.0.0.10', получено '%s'", found.IP)
	}

	// Тест поиска по имени
	foundByName := store.GetResourceByName("test-vm", ResourceDomain)
	if foundByName == nil {
		t.Fatal("ресурс не найден по имени")
	}

	// Тест фильтрации по типу
	domains := store.GetResourcesByType(ResourceDomain)
	if len(domains) != 1 {
		t.Errorf("ожидалось 1 домен, получено %d", len(domains))
	}

	// Тест обновления
	found.Status = StatusStopped
	if err := store.UpdateResource(found); err != nil {
		t.Fatalf("ошибка обновления: %v", err)
	}

	updated := store.GetResource("test-1")
	if updated.Status != StatusStopped {
		t.Errorf("ожидался статус 'stopped', получено '%s'", updated.Status)
	}

	// Тест сохранения
	if err := store.Save(); err != nil {
		t.Fatalf("ошибка сохранения: %v", err)
	}

	// Проверяем что файл существует
	statePath := filepath.Join(tmpDir, StateFileName)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("файл состояния не создан")
	}

	// Тест загрузки из файла
	store2 := NewStore(tmpDir)
	if err := store2.Load(); err != nil {
		t.Fatalf("ошибка загрузки из файла: %v", err)
	}

	loaded := store2.GetResource("test-1")
	if loaded == nil {
		t.Fatal("ресурс не загружен из файла")
	}
	if loaded.Status != StatusStopped {
		t.Errorf("ожидался статус 'stopped' после загрузки, получено '%s'", loaded.Status)
	}

	// Тест удаления
	if err := store.RemoveResource("test-1"); err != nil {
		t.Fatalf("ошибка удаления: %v", err)
	}

	deleted := store.GetResource("test-1")
	if deleted != nil {
		t.Error("ресурс не удалён")
	}
}

func TestHashConfig(t *testing.T) {
	config1 := map[string]string{"name": "test", "cpu": "2"}
	config2 := map[string]string{"name": "test", "cpu": "4"}
	config3 := map[string]string{"name": "test", "cpu": "2"}

	hash1 := HashConfig(config1)
	hash2 := HashConfig(config2)
	hash3 := HashConfig(config3)

	if hash1 == hash2 {
		t.Error("ожидалось разные хеши для разных конфигураций")
	}
	if hash1 != hash3 {
	 t.Error("ожидалось одинаковые хеши для одинаковых конфигураций")
	}
}
