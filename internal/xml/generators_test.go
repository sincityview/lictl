package xml

import (
	"strings"
	"testing"
)

func TestGenerateDomainXML(t *testing.T) {
	cfg := &DomainConfig{
		Name:    "test-vm",
		Memory:  2048,
		VCPUs:   2,
		OSArch:  "x86_64",
		Machine: "pc-q35-8.0",
		Disks: []DomainDisk{
			{
				Type:   "file",
				Device: "disk",
				Source: "/var/lib/libvirt/images/test.qcow2",
				Target: "vda",
				Bus:    "virtio",
				Format: "qcow2",
			},
		},
		Interfaces: []DomainInterface{
			{
				Type:   "network",
				Source: "default",
				Model:  "virtio",
			},
		},
	}

	xml := GenerateDomainXML(cfg)

	if !strings.Contains(xml, "<name>test-vm</name>") {
		t.Error("не найдено имя VM")
	}
	if !strings.Contains(xml, "<memory unit='MiB'>2048</memory>") {
		t.Error("не найдена память")
	}
	if !strings.Contains(xml, "<vcpu placement='static'>2</vcpu>") {
		t.Error("не найдены VCPU")
	}
	if !strings.Contains(xml, "type='qcow2'") {
		t.Error("не найден формат диска")
	}
	if !strings.Contains(xml, "<source network='default'/>") {
		t.Error("не найдена сеть")
	}
	if !strings.Contains(xml, "liblictl:managed") {
		t.Error("не найдены метаданные lictl")
	}
}

func TestGenerateNetworkXML(t *testing.T) {
	cfg := &NetworkConfig{
		Name:   "mgmt",
		Bridge: "virbr1",
		Mode:   "nat",
		Subnet: "10.10.0.0/24",
		DHCP: &DHCPConfig{
			RangeStart: "10.10.0.100",
			RangeEnd:   "10.10.0.200",
		},
	}

	xml := GenerateNetworkXML(cfg)

	if !strings.Contains(xml, "<name>mgmt</name>") {
		t.Error("не найдено имя сети")
	}
	if !strings.Contains(xml, "<bridge name='virbr1'/>") {
		t.Error("не найден мост")
	}
	if !strings.Contains(xml, "<forward mode='nat'/>") {
		t.Error("не найден режим forward")
	}
	if !strings.Contains(xml, "<range start='10.10.0.100' end='10.10.0.200'/>") {
		t.Error("не найден диапазон DHCP")
	}
}

func TestGenerateStoragePoolXML(t *testing.T) {
	cfg := &StoragePoolConfig{
		Name: "default",
		Type: "dir",
		Path: "/var/lib/libvirt/images",
	}

	xml := GenerateStoragePoolXML(cfg)

	if !strings.Contains(xml, "<name>default</name>") {
		t.Error("не найдено имя пула")
	}
	if !strings.Contains(xml, "type='dir'") {
		t.Error("не найден тип пула")
	}
	if !strings.Contains(xml, "<path>/var/lib/libvirt/images</path>") {
		t.Error("не найден путь")
	}
}

func TestGenerateStorageVolumeXML(t *testing.T) {
	cfg := &StorageVolumeConfig{
		Name:     "test.qcow2",
		Capacity: 20 * 1024 * 1024 * 1024, // 20GB
		Format:   "qcow2",
	}

	xml := GenerateStorageVolumeXML(cfg)

	if !strings.Contains(xml, "<name>test.qcow2</name>") {
		t.Error("не найдено имя тома")
	}
	if !strings.Contains(xml, "<format type='qcow2'/>") {
		t.Error("не найден формат")
	}
}

func TestParseDiskSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"20G", 20 * 1024 * 1024 * 1024},
		{"100Gi", 100 * 1024 * 1024 * 1024},
		{"512M", 512 * 1024 * 1024},
		{"1T", 1024 * 1024 * 1024 * 1024},
	}

	for _, test := range tests {
		result, err := ParseDiskSize(test.input)
		if err != nil {
			t.Errorf("ошибка парсинга %s: %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("для %s: ожидалось %d, получено %d", test.input, test.expected, result)
		}
	}
}

func TestFormatDiskSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{20 * 1024 * 1024 * 1024, "20.0G"},
		{100 * 1024 * 1024 * 1024, "100.0G"},
		{512 * 1024 * 1024, "512.0M"},
		{1024 * 1024 * 1024 * 1024, "1.0T"},
	}

	for _, test := range tests {
		result := FormatDiskSize(test.input)
		if result != test.expected {
			t.Errorf("для %d: ожидалось '%s', получено '%s'", test.input, test.expected, result)
		}
	}
}
