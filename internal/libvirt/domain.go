package libvirt

import (
	"fmt"
	"path/filepath"

	"github.com/digitalocean/go-libvirt"
	"github.com/alex/lictl/internal/config"
	"github.com/alex/lictl/internal/xml"
)

// DomainManager управляет виртуальными машинами
type DomainManager struct {
	conn *Connection
}

// NewDomainManager создаёт менеджер доменов
func NewDomainManager(conn *Connection) *DomainManager {
	return &DomainManager{conn: conn}
}

// CreateDomain создаёт виртуальную машину
func (m *DomainManager) CreateDomain(vm config.VMConfig, storagePath, cloudInitPath string) (*DomainResult, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return nil, err
	}

	// Конвертируем пути в абсолютные
	absStoragePath, err := filepath.Abs(storagePath)
	if err != nil {
		absStoragePath = storagePath
	}

	// Формируем диск
	disks := []xml.DomainDisk{
		{
			Type:   "file",
			Device: "disk",
			Source: absStoragePath,
			Target: "vda",
			Bus:    "virtio",
			Format: "qcow2",
		},
	}

	// Cloud-init путь (абсолютный)
	var cloudInitISO string
	if vm.CloudInit != nil && cloudInitPath != "" {
		absCloudInitPath, err := filepath.Abs(cloudInitPath)
		if err != nil {
			absCloudInitPath = cloudInitPath
		}
		cloudInitISO = absCloudInitPath
	}

	// Формируем сети
	var interfaces []xml.DomainInterface
	for _, net := range vm.Networks {
		iface := xml.DomainInterface{
			Type:   "network",
			Source: net.Name,
			Model:  "virtio",
		}
		interfaces = append(interfaces, iface)
	}

	// Если нет сетей, добавляем default
	if len(interfaces) == 0 {
		interfaces = append(interfaces, xml.DomainInterface{
			Type:   "network",
			Source: "default",
			Model:  "virtio",
		})
	}

	cfg := &xml.DomainConfig{
		Name:         vm.Name,
		Memory:       vm.Memory,
		VCPUs:        vm.CPU,
		OSArch:       "x86_64",
		Machine:      "pc-q35-8.0",
		BootDev:      "hd",
		Disks:        disks,
		Interfaces:   interfaces,
		CloudInitISO: cloudInitISO,
	}

	domainXML := xml.GenerateDomainXML(cfg)

	// Определяем домен
	domain, err := m.conn.Libvirt.DomainDefineXML(domainXML)
	if err != nil {
		return nil, fmt.Errorf("ошибка определения домена %s: %w", vm.Name, err)
	}

	// Запускаем домен
	if err := m.conn.Libvirt.DomainCreate(domain); err != nil {
		return nil, fmt.Errorf("ошибка запуска домена %s: %w", vm.Name, err)
	}

	// Устанавливаем автозапуск
	if vm.Autostart {
		if err := m.conn.Libvirt.DomainSetAutostart(domain, 1); err != nil {
			fmt.Printf("предупреждение: не удалось установить автозапуск для %s: %v\n", vm.Name, err)
		}
	}

	return &DomainResult{
		Name: vm.Name,
		UUID: fmt.Sprintf("%x", domain.UUID),
	}, nil
}

// DomainResult результат создания домена
type DomainResult struct {
	Name string
	UUID string
}

// GetDomain возвращает домен по имени
func (m *DomainManager) GetDomain(name string) (libvirt.Domain, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return libvirt.Domain{}, err
	}

	domain, err := m.conn.Libvirt.DomainLookupByName(name)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("домен не найден: %s", name)
	}

	return domain, nil
}

// DomainExists проверяет существование домена
func (m *DomainManager) DomainExists(name string) bool {
	_, err := m.GetDomain(name)
	return err == nil
}

// StartDomain запускает домен
func (m *DomainManager) StartDomain(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	domain, err := m.GetDomain(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.DomainCreate(domain)
}

// StopDomain останавливает домен (graceful shutdown)
func (m *DomainManager) StopDomain(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	domain, err := m.GetDomain(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.DomainShutdownFlags(domain, libvirt.DomainShutdownAcpiPowerBtn)
}

// ForceStopDomain принудительно останавливает домен
func (m *DomainManager) ForceStopDomain(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	domain, err := m.GetDomain(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.DomainDestroyFlags(domain, 0)
}

// RebootDomain перезагружает домен
func (m *DomainManager) RebootDomain(name string) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	domain, err := m.GetDomain(name)
	if err != nil {
		return err
	}

	return m.conn.Libvirt.DomainReboot(domain, 0)
}

// DeleteDomain удаляет домен
func (m *DomainManager) DeleteDomain(name string, force bool) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	domain, err := m.GetDomain(name)
	if err != nil {
		return err
	}

	// Останавливаем если работает
	if force {
		_ = m.conn.Libvirt.DomainDestroyFlags(domain, 0)
	} else {
		_ = m.conn.Libvirt.DomainShutdownFlags(domain, libvirt.DomainShutdownAcpiPowerBtn)
	}

	// Удаляем определение
	return m.conn.Libvirt.DomainUndefine(domain)
}

// ListDomains возвращает список всех доменов
func (m *DomainManager) ListDomains() ([]libvirt.Domain, error) {
	if err := m.conn.EnsureConnect(); err != nil {
		return nil, err
	}

	domains, _, err := m.conn.Libvirt.ConnectListAllDomains(1,
		libvirt.ConnectListDomainsActive|libvirt.ConnectListDomainsInactive)
	if err != nil {
		return nil, err
	}

	return domains, nil
}

// DomainInfo информация о домене
type DomainInfo struct {
	Name      string
	UUID      string
	State     string
	Memory    uint64
	VCPUs     uint16
	XML       string
	Autostart bool
	IPs       []string
}

// GetDomainInfo возвращает информацию о домене
func (m *DomainManager) GetDomainInfo(name string) (*DomainInfo, error) {
	domain, err := m.GetDomain(name)
	if err != nil {
		return nil, err
	}

	xmlStr, err := m.conn.Libvirt.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return nil, err
	}

	autostart, _ := m.conn.Libvirt.DomainGetAutostart(domain)

	return &DomainInfo{
		Name:      name,
		UUID:      fmt.Sprintf("%x", domain.UUID),
		State:     "unknown",
		XML:       xmlStr,
		Autostart: autostart == 1,
	}, nil
}

// SetAutostart устанавливает автозапуск
func (m *DomainManager) SetAutostart(name string, enabled bool) error {
	if err := m.conn.EnsureConnect(); err != nil {
		return err
	}

	domain, err := m.GetDomain(name)
	if err != nil {
		return err
	}

	val := int32(0)
	if enabled {
		val = 1
	}

	return m.conn.Libvirt.DomainSetAutostart(domain, val)
}
