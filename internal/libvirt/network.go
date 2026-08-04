package libvirt

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
	"github.com/sincityview/lictl/internal/xml"
)

// NetworkManager управляет виртуальными сетями
type NetworkManager struct {
	conn *Connection
}

// NewNetworkManager создаёт менеджер сетей
func NewNetworkManager(conn *Connection) *NetworkManager {
	return &NetworkManager{conn: conn}
}

// CreateNetwork создаёт сеть
func (m *NetworkManager) CreateNetwork(cfg *xml.NetworkConfig) (string, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return "", err
	}

	// Генерируем XML
	networkXML := xml.GenerateNetworkXML(cfg)

	// Определяем сеть
	network, err := m.conn.Libvirt.NetworkDefineXML(networkXML)
	if err != nil {
		return "", fmt.Errorf("ошибка определения сети %s: %w", cfg.Name, err)
	}

	// Запускаем сеть
	if err := m.conn.Libvirt.NetworkCreate(network); err != nil {
		return "", fmt.Errorf("ошибка запуска сети %s: %w", cfg.Name, err)
	}

	// Устанавливаем автозапуск
	if cfg.Autostart {
		if err := m.conn.Libvirt.NetworkSetAutostart(network, 1); err != nil {
			fmt.Printf("предупреждение: не удалось установить автозапуск для сети %s: %v\n", cfg.Name, err)
		}
	}

	return cfg.Name, nil
}

// GetNetwork возвращает сеть по имени
func (m *NetworkManager) GetNetwork(name string) (libvirt.Network, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return libvirt.Network{}, err
	}

	network, err := m.conn.Libvirt.NetworkLookupByName(name)
	if err != nil {
		return libvirt.Network{}, fmt.Errorf("сеть не найдена: %s", name)
	}

	return network, nil
}

// NetworkExists проверяет существование сети
func (m *NetworkManager) NetworkExists(name string) bool {
	_, err := m.GetNetwork(name)
	return err == nil
}

// StartNetwork запускает сеть
func (m *NetworkManager) StartNetwork(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	network, err := m.GetNetwork(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.NetworkCreate(network)
}

// StopNetwork останавливает сеть
func (m *NetworkManager) StopNetwork(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	network, err := m.GetNetwork(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.NetworkDestroy(network)
}

// DeleteNetwork удаляет сеть
func (m *NetworkManager) DeleteNetwork(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	network, err := m.GetNetwork(name)
	if err != nil {
		return err
	}

	// Останавливаем если запущена
	_ = m.conn.Libvirt.NetworkDestroy(network)

	// Удаляем определение
	return m.conn.Libvirt.NetworkUndefine(network)
}

// ListNetworks возвращает список всех сетей
func (m *NetworkManager) ListNetworks() ([]libvirt.Network, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return nil, err
	}

	networks, _, err := m.conn.Libvirt.ConnectListAllNetworks(1,
		libvirt.ConnectListNetworksActive|libvirt.ConnectListNetworksInactive)
	if err != nil {
		return nil, err
	}

	return networks, nil
}

// NetworkInfo информация о сети
type NetworkInfo struct {
	Name      string
	XML       string
	Active    bool
	Autostart bool
}

// GetNetworkInfo возвращает информацию о сети
func (m *NetworkManager) GetNetworkInfo(name string) (*NetworkInfo, error) {
	network, err := m.GetNetwork(name)
	if err != nil {
		return nil, err
	}

	xmlStr, err := m.conn.Libvirt.NetworkGetXMLDesc(network, 0)
	if err != nil {
		return nil, err
	}

	autostart, _ := m.conn.Libvirt.NetworkGetAutostart(network)

	return &NetworkInfo{
		Name:      name,
		XML:       xmlStr,
		Active:    true,
		Autostart: autostart == 1,
	}, nil
}

// SetAutostart устанавливает автозапуск
func (m *NetworkManager) SetAutostart(name string, enabled bool) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	network, err := m.GetNetwork(name)
	if err != nil {
		return err
	}

	val := int32(0)
	if enabled {
		val = 1
	}

	return m.conn.Libvirt.NetworkSetAutostart(network, val)
}
