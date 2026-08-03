package plan

import (
	"fmt"

	"github.com/alex/lictl/internal/config"
	libvirtclient "github.com/alex/lictl/internal/libvirt"
	"github.com/alex/lictl/internal/state"
)

// Importer импортирует существующие ресурсы
type Importer struct {
	conn  *libvirtclient.Connection
	store *state.Store
}

// NewImporter создаёт импортёр
func NewImporter(conn *libvirtclient.Connection, store *state.Store) *Importer {
	return &Importer{
		conn:  conn,
		store: store,
	}
}

// ImportAll импортирует все существующие ресурсы
func (i *Importer) ImportAll(cfg *config.Config) (*ImportResult, error) {
	result := &ImportResult{}

	// Импортируем Storage
	storageCount, err := i.importStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка импорта storage: %w", err)
	}
	result.Storage = storageCount

	// Импортируем Networks
	networkCount, err := i.importNetworks(cfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка импорта networks: %w", err)
	}
	result.Networks = networkCount

	// Импортируем Domains
	domainCount, err := i.importDomains(cfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка импорта domains: %w", err)
	}
	result.Domains = domainCount

	// Сохраняем state
	if err := i.store.Save(); err != nil {
		return nil, fmt.Errorf("ошибка сохранения состояния: %w", err)
	}

	return result, nil
}

// ImportResult результат импорта
type ImportResult struct {
	Storage  int
	Networks int
	Domains  int
}

// importStorage импортирует пулы хранения
func (i *Importer) importStorage(cfg *config.Config) (int, error) {
	manager := libvirtclient.NewStorageManager(i.conn)

	pools, err := manager.ListPools()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, poolName := range pools {
		// Проверяем не импортирован ли уже
		if i.store.GetResourceByName(poolName, state.ResourceStorage) != nil {
			continue
		}

		resource := state.NewResource(poolName, poolName, state.ResourceStorage)
		resource.UpdateStatus(state.StatusRunning)
		i.store.AddResource(resource)
		count++
	}

	return count, nil
}

// importNetworks импортирует сети
func (i *Importer) importNetworks(cfg *config.Config) (int, error) {
	manager := libvirtclient.NewNetworkManager(i.conn)

	networks, err := manager.ListNetworks()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, network := range networks {
		name := network.Name

		// Проверяем не импортирована ли уже
		if i.store.GetResourceByName(name, state.ResourceNetwork) != nil {
			continue
		}

		resource := state.NewResource(name, name, state.ResourceNetwork)
		resource.UpdateStatus(state.StatusRunning)
		i.store.AddResource(resource)
		count++
	}

	return count, nil
}

// importDomains импортирует виртуальные машины
func (i *Importer) importDomains(cfg *config.Config) (int, error) {
	manager := libvirtclient.NewDomainManager(i.conn)

	domains, err := manager.ListDomains()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, domain := range domains {
		name := domain.Name

		// Проверяем не импортирован ли уже
		if i.store.GetResourceByName(name, state.ResourceDomain) != nil {
			continue
		}

		resource := state.NewResource(name, name, state.ResourceDomain)
		resource.UpdateStatus(state.StatusRunning)
		resource.SetLibvirtID(fmt.Sprintf("%x", domain.UUID))
		i.store.AddResource(resource)
		count++
	}

	return count, nil
}
