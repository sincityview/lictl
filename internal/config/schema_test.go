package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Создаём временный файл
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "lictl.yaml")

	cfgContent := `provider:
  libvirt:
    uri: "qemu:///system"

resources:
  storage:
    - name: default
      type: dir
      path: /var/lib/libvirt/images
      autostart: true

  networks:
    - name: mgmt
      mode: nat
      subnet: 10.10.0.0/24
      dhcp:
        start: 10.10.0.100
        end: 10.10.0.200
      autostart: true

  vms:
    - name: test-vm
      cpu: 2
      memory: 2048
      networks:
        - name: mgmt
`

	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("ошибка создания файла: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("ошибка загрузки: %v", err)
	}

	if cfg.Provider.Libvirt.URI != "qemu:///system" {
		t.Errorf("ожидался URI 'qemu:///system', получено '%s'", cfg.Provider.Libvirt.URI)
	}

	if len(cfg.Resources.Storage) != 1 {
		t.Errorf("ожидался 1 пул, получено %d", len(cfg.Resources.Storage))
	}

	if len(cfg.Resources.Networks) != 1 {
		t.Errorf("ожидалась 1 сеть, получено %d", len(cfg.Resources.Networks))
	}

	if len(cfg.Resources.VMs) != 1 {
		t.Errorf("ожидалась 1 VM, получено %d", len(cfg.Resources.VMs))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Provider: Provider{
					Libvirt: LibvirtProvider{URI: "qemu:///system"},
				},
				Resources: Resources{
					VMs: []VMConfig{
						{Name: "test", CPU: 1, Memory: 1024},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing URI",
			cfg: Config{
				Provider: Provider{
					Libvirt: LibvirtProvider{URI: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "missing VM name",
			cfg: Config{
				Provider: Provider{
					Libvirt: LibvirtProvider{URI: "qemu:///system"},
				},
				Resources: Resources{
					VMs: []VMConfig{
						{Name: "", CPU: 1, Memory: 1024},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "zero CPU",
			cfg: Config{
				Provider: Provider{
					Libvirt: LibvirtProvider{URI: "qemu:///system"},
				},
				Resources: Resources{
					VMs: []VMConfig{
						{Name: "test", CPU: 0, Memory: 1024},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSubnets(t *testing.T) {
	tests := []struct {
		name    string
		networks []NetworkConfig
		wantErr bool
	}{
		{
			name: "unique subnets",
			networks: []NetworkConfig{
				{Name: "net1", Subnet: "10.0.0.0/24"},
				{Name: "net2", Subnet: "192.168.0.0/24"},
			},
			wantErr: false,
		},
		{
			name: "duplicate subnets",
			networks: []NetworkConfig{
				{Name: "net1", Subnet: "10.0.0.0/24"},
				{Name: "net2", Subnet: "10.0.0.0/24"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubnets(tt.networks)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubnets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
