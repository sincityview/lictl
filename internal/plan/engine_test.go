package plan

import (
	"testing"

	"github.com/alex/lictl/internal/config"
	"github.com/alex/lictl/internal/state"
)

func TestEnginePlan(t *testing.T) {
	// Создаём временное хранилище
	store := state.NewStore(t.TempDir())
	if err := store.Load(); err != nil {
		t.Fatalf("ошибка загрузки: %v", err)
	}

	engine := NewEngine(store)

	cfg := &config.Config{
		Provider: config.Provider{
			Libvirt: config.LibvirtProvider{URI: "qemu:///system"},
		},
		Resources: config.Resources{
			Storage: []config.StorageConfig{
				{Name: "default", Type: "dir", Path: "/var/lib/libvirt/images"},
			},
			Networks: []config.NetworkConfig{
				{Name: "mgmt", Mode: "nat", Subnet: "10.0.0.0/24"},
			},
			VMs: []config.VMConfig{
				{Name: "test-vm", CPU: 2, Memory: 2048},
			},
		},
	}

	plan, err := engine.Plan(cfg)
	if err != nil {
		t.Fatalf("ошибка генерации плана: %v", err)
	}

	// Проверяем что план содержит все ресурсы
	if plan.Summary.Create != 3 {
		t.Errorf("ожидалось 3 создания, получено %d", plan.Summary.Create)
	}

	// Проверяем порядок (Storage → Network → VM)
	storageIdx := -1
	networkIdx := -1
	vmIdx := -1

	for i, change := range plan.Changes {
		switch change.ResourceType {
		case state.ResourceStorage:
			storageIdx = i
		case state.ResourceNetwork:
			networkIdx = i
		case state.ResourceDomain:
			vmIdx = i
		}
	}

	if storageIdx >= networkIdx {
		t.Error("Storage должен быть перед Network")
	}
	if networkIdx >= vmIdx {
		t.Error("Network должен быть перед VM")
	}
}

func TestEnginePlanNoChanges(t *testing.T) {
	store := state.NewStore(t.TempDir())
	if err := store.Load(); err != nil {
		t.Fatalf("ошибка загрузки: %v", err)
	}

	engine := NewEngine(store)

	cfg := &config.Config{
		Provider: config.Provider{
			Libvirt: config.LibvirtProvider{URI: "qemu:///system"},
		},
		Resources: config.Resources{},
	}

	plan, err := engine.Plan(cfg)
	if err != nil {
		t.Fatalf("ошибка генерации плана: %v", err)
	}

	if plan.Summary.Total != 0 {
		t.Errorf("ожидалось 0 изменений, получено %d", plan.Summary.Total)
	}
}

func TestSortChanges(t *testing.T) {
	changes := []Change{
		{ResourceType: state.ResourceDomain, Name: "vm1"},
		{ResourceType: state.ResourceStorage, Name: "pool1"},
		{ResourceType: state.ResourceNetwork, Name: "net1"},
		{ResourceType: state.ResourceDomain, Name: "vm2"},
	}

	sorted := SortChanges(changes)

	// Проверяем порядок
	expectedOrder := []string{"pool1", "net1", "vm1", "vm2"}
	for i, expected := range expectedOrder {
		if sorted[i].Name != expected {
			t.Errorf("позиция %d: ожидалось '%s', получено '%s'", i, expected, sorted[i].Name)
		}
	}
}
