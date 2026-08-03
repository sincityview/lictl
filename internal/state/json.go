package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	StateFileName = ".lictl-state.json"
	StateVersion  = 1
)

// Store хранилище состояния на основе JSON-файла
type Store struct {
	path  string
	state *StateFile
}

// NewStore создаёт новое хранилище
func NewStore(dir string) *Store {
	return &Store{
		path: filepath.Join(dir, StateFileName),
	}
}

// Load загружает состояние из файла
func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создаём пустое состояние
			s.state = &StateFile{
				Version:   StateVersion,
				Resources: []Resource{},
				Metadata: StateMetadata{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}
			return nil
		}
		return fmt.Errorf("ошибка чтения состояния: %w", err)
	}

	var state StateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("ошибка парсинга состояния: %w", err)
	}

	s.state = &state
	return nil
}

// Save сохраняет состояние в файл
func (s *Store) Save() error {
	s.state.Metadata.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации состояния: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи состояния: %w", err)
	}

	return nil
}

// GetResource возвращает ресурс по ID
func (s *Store) GetResource(id string) *Resource {
	for i := range s.state.Resources {
		if s.state.Resources[i].ID == id {
			return &s.state.Resources[i]
		}
	}
	return nil
}

// GetResourceByName возвращает ресурс по имени и типу
func (s *Store) GetResourceByName(name string, rtype ResourceType) *Resource {
	for i := range s.state.Resources {
		if s.state.Resources[i].Name == name && s.state.Resources[i].Type == rtype {
			return &s.state.Resources[i]
		}
	}
	return nil
}

// GetResourcesByType возвращает все ресурсы указанного типа
func (s *Store) GetResourcesByType(rtype ResourceType) []Resource {
	var result []Resource
	for _, r := range s.state.Resources {
		if r.Type == rtype {
			result = append(result, r)
		}
	}
	return result
}

// GetAllResources возвращает все ресурсы
func (s *Store) GetAllResources() []Resource {
	return s.state.Resources
}

// AddResource добавляет новый ресурс
func (s *Store) AddResource(r *Resource) {
	s.state.Resources = append(s.state.Resources, *r)
}

// UpdateResource обновляет существующий ресурс
func (s *Store) UpdateResource(r *Resource) error {
	for i := range s.state.Resources {
		if s.state.Resources[i].ID == r.ID {
			s.state.Resources[i] = *r
			return nil
		}
	}
	return fmt.Errorf("ресурс не найден: %s", r.ID)
}

// RemoveResource удаляет ресурс
func (s *Store) RemoveResource(id string) error {
	for i := range s.state.Resources {
		if s.state.Resources[i].ID == id {
			s.state.Resources = append(s.state.Resources[:i], s.state.Resources[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("ресурс не найден: %s", id)
}

// SetProviderURI устанавливает URI провайдера
func (s *Store) SetProviderURI(uri string) {
	s.state.Metadata.ProviderURI = uri
}

// GetProviderURI возвращает URI провайдера
func (s *Store) GetProviderURI() string {
	return s.state.Metadata.ProviderURI
}

// HashConfig вычисляет хеш конфигурации
func HashConfig(config interface{}) string {
	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8])
}

// GetStateFileInfo возвращает информацию о файле состояния
func (s *Store) GetStateFileInfo() (os.FileInfo, error) {
	return os.Stat(s.path)
}

// Backup создаёт резервную копию состояния
func (s *Store) Backup() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	backupPath := s.path + ".backup"
	return os.WriteFile(backupPath, data, 0644)
}

// Clear очищает все ресурсы
func (s *Store) Clear() {
	s.state.Resources = []Resource{}
}
