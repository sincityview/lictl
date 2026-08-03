package plan

import (
	"fmt"
	"sort"

	"github.com/alex/lictl/internal/config"
	"github.com/alex/lictl/internal/state"
)

// ChangeType тип изменения
type ChangeType string

const (
	Create ChangeType = "create"
	Update ChangeType = "update"
	Delete ChangeType = "delete"
	NoOp   ChangeType = "no-op"
)

// Change описание изменения
type Change struct {
	Type       ChangeType
	ResourceType state.ResourceType
	Name       string
	Current    *state.Resource
	Desired    interface{} // config конфигурация
	Details    string
}

// Plan результат сравнения
type Plan struct {
	Changes []Change
	Summary Summary
}

// Summary статистика плана
type Summary struct {
	Create int
	Update int
	Delete int
	NoOp   int
	Total  int
}

// Engine движок плана
type Engine struct {
	store *state.Store
}

// NewEngine создаёт новый движок плана
func NewEngine(store *state.Store) *Engine {
	return &Engine{store: store}
}

// Plan генерирует план изменений
func (e *Engine) Plan(cfg *config.Config) (*Plan, error) {
	plan := &Plan{}

	// Расширяем VM с диапазонами
	expandedVMs := config.ExpandAllVMs(cfg.Resources.VMs)

	// Планируем Storage
	storageChanges := e.planStorage(cfg.Resources.Storage)
	plan.Changes = append(plan.Changes, storageChanges...)

	// Планируем Networks
	networkChanges := e.planNetworks(cfg.Resources.Networks)
	plan.Changes = append(plan.Changes, networkChanges...)

	// Планируем VMs
	vmChanges := e.planVMs(expandedVMs)
	plan.Changes = append(plan.Changes, vmChanges...)

	// Считаем статистику
	for _, change := range plan.Changes {
		switch change.Type {
		case Create:
			plan.Summary.Create++
		case Update:
			plan.Summary.Update++
		case Delete:
			plan.Summary.Delete++
		case NoOp:
			plan.Summary.NoOp++
		}
	}
	plan.Summary.Total = len(plan.Changes)

	return plan, nil
}

// planStorage планирует изменения для пулов хранения
func (e *Engine) planStorage(storageConfigs []config.StorageConfig) []Change {
	var changes []Change

	for _, cfg := range storageConfigs {
		existing := e.store.GetResourceByName(cfg.Name, state.ResourceStorage)

		if existing == nil {
			changes = append(changes, Change{
				Type:         Create,
				ResourceType: state.ResourceStorage,
				Name:         cfg.Name,
				Desired:      cfg,
				Details:      fmt.Sprintf("создать пул %s (%s)", cfg.Name, cfg.Type),
			})
		} else {
			// Проверяем нужно ли обновление
			hash := state.HashConfig(cfg)
			if existing.ConfigHash != hash {
				changes = append(changes, Change{
					Type:         Update,
					ResourceType: state.ResourceStorage,
					Name:         cfg.Name,
					Current:      existing,
					Desired:      cfg,
					Details:      fmt.Sprintf("обновить пул %s", cfg.Name),
				})
			} else {
				changes = append(changes, Change{
					Type:         NoOp,
					ResourceType: state.ResourceStorage,
					Name:         cfg.Name,
					Current:      existing,
					Details:      fmt.Sprintf("пул %s без изменений", cfg.Name),
				})
			}
		}
	}

	// Проверяем пулы которые нужно удалить
	existingPools := e.store.GetResourcesByType(state.ResourceStorage)
	for _, existing := range existingPools {
		found := false
		for _, cfg := range storageConfigs {
			if cfg.Name == existing.Name {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, Change{
				Type:         Delete,
			 ResourceType: state.ResourceStorage,
				Name:         existing.Name,
				Current:      &existing,
				Details:      fmt.Sprintf("удалить пул %s", existing.Name),
			})
		}
	}

	return changes
}

// planNetworks планирует изменения для сетей
func (e *Engine) planNetworks(networkConfigs []config.NetworkConfig) []Change {
	var changes []Change

	for _, cfg := range networkConfigs {
		existing := e.store.GetResourceByName(cfg.Name, state.ResourceNetwork)

		if existing == nil {
			changes = append(changes, Change{
				Type:         Create,
				ResourceType: state.ResourceNetwork,
				Name:         cfg.Name,
				Desired:      cfg,
				Details:      fmt.Sprintf("создать сеть %s (%s)", cfg.Name, cfg.Mode),
			})
		} else {
			hash := state.HashConfig(cfg)
			if existing.ConfigHash != hash {
				changes = append(changes, Change{
					Type:         Update,
					ResourceType: state.ResourceNetwork,
					Name:         cfg.Name,
					Current:      existing,
					Desired:      cfg,
					Details:      fmt.Sprintf("обновить сеть %s", cfg.Name),
				})
			} else {
				changes = append(changes, Change{
					Type:         NoOp,
					ResourceType: state.ResourceNetwork,
					Name:         cfg.Name,
					Current:      existing,
					Details:      fmt.Sprintf("сеть %s без изменений", cfg.Name),
				})
			}
		}
	}

	// Проверяем сети для удаления
	existingNetworks := e.store.GetResourcesByType(state.ResourceNetwork)
	for _, existing := range existingNetworks {
		found := false
		for _, cfg := range networkConfigs {
			if cfg.Name == existing.Name {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, Change{
				Type:         Delete,
				ResourceType: state.ResourceNetwork,
				Name:         existing.Name,
				Current:      &existing,
				Details:      fmt.Sprintf("удалить сеть %s", existing.Name),
			})
		}
	}

	return changes
}

// planVMs планирует изменения для VM
func (e *Engine) planVMs(vmConfigs []config.VMConfig) []Change {
	var changes []Change

	for _, cfg := range vmConfigs {
		existing := e.store.GetResourceByName(cfg.Name, state.ResourceDomain)

		if existing == nil {
			changes = append(changes, Change{
				Type:         Create,
				ResourceType: state.ResourceDomain,
				Name:         cfg.Name,
				Desired:      cfg,
				Details:      fmt.Sprintf("создать VM %s (CPU: %d, RAM: %dMiB)", cfg.Name, cfg.CPU, cfg.Memory),
			})
		} else {
			hash := state.HashConfig(cfg)
			if existing.ConfigHash != hash {
				changes = append(changes, Change{
					Type:         Update,
					ResourceType: state.ResourceDomain,
					Name:         cfg.Name,
					Current:      existing,
					Desired:      cfg,
					Details:      fmt.Sprintf("обновить VM %s", cfg.Name),
				})
			} else {
				changes = append(changes, Change{
					Type:         NoOp,
					ResourceType: state.ResourceDomain,
					Name:         cfg.Name,
					Current:      existing,
					Details:      fmt.Sprintf("VM %s без изменений", cfg.Name),
				})
			}
		}
	}

	// Проверяем VM для удаления
	existingVMs := e.store.GetResourcesByType(state.ResourceDomain)
	for _, existing := range existingVMs {
		found := false
		for _, cfg := range vmConfigs {
			if cfg.Name == existing.Name {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, Change{
				Type:         Delete,
				ResourceType: state.ResourceDomain,
				Name:         existing.Name,
				Current:      &existing,
				Details:      fmt.Sprintf("удалить VM %s", existing.Name),
			})
		}
	}

	return changes
}

// SortChanges сортирует изменения в порядке зависимостей
// Storage → Networks → VMs
func SortChanges(changes []Change) []Change {
	result := make([]Change, len(changes))
	copy(result, changes)

	sort.SliceStable(result, func(i, j int) bool {
		priorityI := resourcePriority(result[i].ResourceType)
		priorityJ := resourcePriority(result[j].ResourceType)

		if priorityI != priorityJ {
			return priorityI < priorityJ
		}

		// Delete идёт перед Create
		if result[i].Type != result[j].Type {
			if result[i].Type == Delete {
				return true
			}
			if result[j].Type == Delete {
				return false
			}
		}

		return result[i].Name < result[j].Name
	})

	return result
}

// resourcePriority приоритет ресурса для порядка применения
func resourcePriority(rtype state.ResourceType) int {
	switch rtype {
	case state.ResourceStorage:
		return 1
	case state.ResourceNetwork:
		return 2
	case state.ResourceDomain:
		return 3
	default:
		return 99
	}
}
