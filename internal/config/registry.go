package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	registryDir  = ".config/lictl"
	registryFile = "projects.json"
)

// Project запись в реестре проектов
type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Registry реестр проектов lictl
type Registry struct {
	Projects []Project `json:"projects"`
	path     string
}

// getRegistryPath возвращает путь к файлу реестра
func getRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("не удалось определить HOME: %w", err)
	}
	return filepath.Join(home, registryDir, registryFile), nil
}

// LoadRegistry загружает реестр из файла
func LoadRegistry() (*Registry, error) {
	path, err := getRegistryPath()
	if err != nil {
		return nil, err
	}

	r := &Registry{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			r.Projects = []Project{}
			return r, nil
		}
		return nil, fmt.Errorf("ошибка чтения реестра: %w", err)
	}

	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("ошибка парсинга реестра: %w", err)
	}

	return r, nil
}

// SaveRegistry сохраняет реестр в файл
func (r *Registry) SaveRegistry() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации реестра: %w", err)
	}

	return os.WriteFile(r.path, data, 0644)
}

// FindProject ищет проект по имени
func (r *Registry) FindProject(name string) *Project {
	for i := range r.Projects {
		if r.Projects[i].Name == name {
			return &r.Projects[i]
		}
	}
	return nil
}

// RegisterProject добавляет или обновляет проект
func (r *Registry) RegisterProject(name, path string) error {
	// Проверяем существует ли
	for i := range r.Projects {
		if r.Projects[i].Name == name {
			r.Projects[i].Path = path
			return r.SaveRegistry()
		}
	}
	r.Projects = append(r.Projects, Project{Name: name, Path: path})
	return r.SaveRegistry()
}

// RemoveProject удаляет проект из реестра
func (r *Registry) RemoveProject(name string) error {
	for i := range r.Projects {
		if r.Projects[i].Name == name {
			r.Projects = append(r.Projects[:i], r.Projects[i+1:]...)
			return r.SaveRegistry()
		}
	}
	return nil
}
