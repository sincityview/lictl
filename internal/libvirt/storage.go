package libvirt

import (
	"fmt"

	"github.com/alex/lictl/internal/xml"
)

// StorageManager управляет пулами хранения и томами
type StorageManager struct {
	conn *Connection
}

// NewStorageManager создаёт менеджер хранилища
func NewStorageManager(conn *Connection) *StorageManager {
	return &StorageManager{conn: conn}
}

// CreatePool создаёт пул хранения
func (m *StorageManager) CreatePool(cfg *xml.StoragePoolConfig) (string, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return "", err
	}

	// Генерируем XML
	poolXML := xml.GenerateStoragePoolXML(cfg)

	// Определяем пул
	pool, err := m.conn.Libvirt.StoragePoolDefineXML(poolXML, 0)
	if err != nil {
		return "", fmt.Errorf("ошибка определения пула %s: %w", cfg.Name, err)
	}

	// Строим пул
	if err := m.conn.Libvirt.StoragePoolBuild(pool, 0); err != nil {
		return "", fmt.Errorf("ошибка построения пула %s: %w", cfg.Name, err)
	}

	// Запускаем пул
	if err := m.conn.Libvirt.StoragePoolCreate(pool, 0); err != nil {
		return "", fmt.Errorf("ошибка запуска пула %s: %w", cfg.Name, err)
	}

	// Устанавливаем автозапуск
	if cfg.Autostart {
		if err := m.conn.Libvirt.StoragePoolSetAutostart(pool, 1); err != nil {
			fmt.Printf("предупреждение: не удалось установить автозапуск для пула %s: %v\n", cfg.Name, err)
		}
	}

	return cfg.Name, nil
}

// PoolExists проверяет существование пула
func (m *StorageManager) PoolExists(name string) bool {
	if err := m.conn.EnsureConnect(); err != nil {
		return false
	}

	_, err := m.conn.Libvirt.StoragePoolLookupByName(name)
	return err == nil
}

// DeletePool удаляет пул и его содержимое
func (m *StorageManager) DeletePool(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	pool, err := m.conn.Libvirt.StoragePoolLookupByName(name)
	if err != nil {
		return nil
	}

	// Останавливаем если запущен
	_ = m.conn.Libvirt.StoragePoolDestroy(pool)

	// Удаляем определение
	return m.conn.Libvirt.StoragePoolUndefine(pool)
}

// ListPools возвращает список всех пулов
func (m *StorageManager) ListPools() ([]string, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return nil, err
	}

	pools, _, err := m.conn.Libvirt.ConnectListAllStoragePools(1, 0)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(pools))
	for i, p := range pools {
		result[i] = p.Name
	}
	return result, nil
}

// RefreshPool обновляет пул
func (m *StorageManager) RefreshPool(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	pool, err := m.conn.Libvirt.StoragePoolLookupByName(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.StoragePoolRefresh(pool, 0)
}
