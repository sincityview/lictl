package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config — корневая структура lictl.yaml
type Config struct {
	Provider  Provider  `yaml:"provider"`
	Resources Resources `yaml:"resources"`
}

// Provider — настройки провайдера
type Provider struct {
	Libvirt LibvirtProvider `yaml:"libvirt"`
}

// LibvirtProvider — настройки подключения к libvirt
type LibvirtProvider struct {
	URI string `yaml:"uri"`
}

// Resources — все управляемые ресурсы
type Resources struct {
	BaseImages []BaseImageConfig `yaml:"base_images,omitempty"`
	Storage    []StorageConfig   `yaml:"storage"`
	Networks   []NetworkConfig   `yaml:"networks"`
	VMs        []VMConfig        `yaml:"vms"`
}

// BaseImageConfig — базовый образ для VM
type BaseImageConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path,omitempty"` // локальный путь
	URL  string `yaml:"url,omitempty"`  // URL для скачивания
}

// StorageConfig — пул хранения
type StorageConfig struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Path      string `yaml:"path,omitempty"`
	VgName    string `yaml:"vg_name,omitempty"`
	Autostart bool   `yaml:"autostart"`
}

// NetworkConfig — виртуальная сеть
type NetworkConfig struct {
	Name      string      `yaml:"name"`
	Mode      string      `yaml:"mode"`
	Bridge    string      `yaml:"bridge,omitempty"`
	Subnet    string      `yaml:"subnet"`
	DHCP      *DHCPConfig `yaml:"dhcp,omitempty"`
	DNS       *DNSConfig  `yaml:"dns,omitempty"`
	Autostart bool        `yaml:"autostart"`
}

// DHCPConfig — настройки DHCP
type DHCPConfig struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

// DNSConfig — настройки DNS
type DNSConfig struct {
	Enable bool `yaml:"enable"`
}

// VMConfig — виртуальная машина
type VMConfig struct {
	Name        string      `yaml:"name"`
	BaseImage   string      `yaml:"base_image,omitempty"`
	StoragePool string      `yaml:"storage,omitempty"`
	CPU         int         `yaml:"cpu"`
	Memory      int         `yaml:"memory"`
	Disk        string      `yaml:"disk,omitempty"`
	Networks    []VMNetwork `yaml:"networks,omitempty"`
	CloudInit   *CloudInit  `yaml:"cloud_init,omitempty"`
	Autostart   bool        `yaml:"autostart"`
}

// VMNetwork — сеть VM (просто имя сети)
type VMNetwork string

// CloudInit — cloud-init конфигурация
type CloudInit struct {
	Hostname string   `yaml:"hostname,omitempty"`
	Users    []CIUser `yaml:"users,omitempty"`
	Packages []string `yaml:"packages,omitempty"`
	RunCmd   []string `yaml:"runcmd,omitempty"`
}

// CIUser — пользователь cloud-init
type CIUser struct {
	Name          string   `yaml:"name"`
	SSHPublicKeys []string `yaml:"ssh_authorized_keys,omitempty"`
	Sudo          bool     `yaml:"sudo,omitempty"`
	Shell         string   `yaml:"shell,omitempty"`
	LockPassword  bool     `yaml:"lock_password,omitempty"`
}

// LoadConfig загружает и парсит lictl.yaml
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга %s: %w", path, err)
	}

	return &cfg, nil
}

// Validate проверяет валидность конфигурации
func (c *Config) Validate() error {
	if c.Provider.Libvirt.URI == "" {
		return fmt.Errorf("provider.libvirt.uri обязателен")
	}

	// Проверяем base_images
	seen := make(map[string]bool)
	for _, bi := range c.Resources.BaseImages {
		if bi.Name == "" {
			return fmt.Errorf("base_image без имени")
		}
		if seen[bi.Name] {
			return fmt.Errorf("дублирующееся имя base_image: %s", bi.Name)
		}
		seen[bi.Name] = true

		if bi.Path == "" && bi.URL == "" {
			return fmt.Errorf("base_image %s: нужен path или url", bi.Name)
		}
	}

	// Проверяем уникальность имён storage
	seen = make(map[string]bool)
	for _, s := range c.Resources.Storage {
		if s.Name == "" {
			return fmt.Errorf("пул хранения без имени")
		}
		if seen[s.Name] {
			return fmt.Errorf("дублирующееся имя пула: %s", s.Name)
		}
		seen[s.Name] = true

		if s.Type == "" {
			return fmt.Errorf("пул %s: тип обязателен", s.Name)
		}
	}

	seen = make(map[string]bool)
	for _, n := range c.Resources.Networks {
		if n.Name == "" {
			return fmt.Errorf("сеть без имени")
		}
		if seen[n.Name] {
			return fmt.Errorf("дублирующееся имя сети: %s", n.Name)
		}
		seen[n.Name] = true

		if n.Mode == "" {
			return fmt.Errorf("сеть %s: режим обязателен", n.Name)
		}
		if n.Subnet == "" {
			return fmt.Errorf("сеть %s: подсеть обязательна", n.Name)
		}
	}

	seen = make(map[string]bool)
	for _, vm := range c.Resources.VMs {
		if vm.Name == "" {
			return fmt.Errorf("VM без имени")
		}
		if seen[vm.Name] {
			return fmt.Errorf("дублирующееся имя VM: %s", vm.Name)
		}
		seen[vm.Name] = true

		if vm.CPU <= 0 {
			return fmt.Errorf("VM %s: cpu должен быть > 0", vm.Name)
		}
		if vm.Memory <= 0 {
			return fmt.Errorf("VM %s: memory должен быть > 0", vm.Name)
		}

		// Проверяем base_image ссылку
		if vm.BaseImage != "" {
			if !isPath(vm.BaseImage) && !c.baseImageExists(vm.BaseImage) {
				return fmt.Errorf("VM %s: base_image '%s' не найден в base_images", vm.Name, vm.BaseImage)
			}
		}
	}

	return nil
}

// baseImageExists проверяет existence base_image по имени
func (c *Config) baseImageExists(name string) bool {
	for _, bi := range c.Resources.BaseImages {
		if bi.Name == name {
			return true
		}
	}
	return false
}

// ResolveBaseImage возвращает путь к base_image по имени или пути
func (c *Config) ResolveBaseImage(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	if isPath(ref) {
		return ref, nil
	}
	for _, bi := range c.Resources.BaseImages {
		if bi.Name == ref {
			if bi.Path != "" {
				return bi.Path, nil
			}
			return "", fmt.Errorf("base_image %s: путь не указан (url загрузка не реализована)", ref)
		}
	}
	return "", fmt.Errorf("base_image не найден: %s", ref)
}

// isPath проверяет является ли строка путём (а не именем)
func isPath(s string) bool {
	return len(s) > 0 && (s[0] == '/' || s[0] == '~' || s[0] == '.')
}
