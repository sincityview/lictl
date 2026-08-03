package xml

import (
	"fmt"
	"strings"
)

// DomainConfig конфигурация домена для генерации XML
type DomainConfig struct {
	Name        string
	UUID        string
	Memory      int    // MiB
	VCPUs       int
	VCPUPlacement string // "static" или "auto"
	OSArch      string
	Machine     string
	BootDev     string
	Disks       []DomainDisk
	Interfaces  []DomainInterface
	Graphics    *DomainGraphics
	CloudInitISO string
	Autostart   bool
}

// DomainDisk диск домена
type DomainDisk struct {
	Type     string // file, volume, block
	Device   string // disk, cdrom
	Source   string // путь к файлу или pool/volume
	Target   string // dev (vda, sda, etc.)
	Bus      string // virtio, sata, ide
	Format   string // qcow2, raw
	ReadOnly bool
}

// DomainInterface сетевой интерфейс
type DomainInterface struct {
	Type   string // network, bridge, direct
	Source string // имя сети или моста
	Model  string // virtio, e1000, etc.
	MAC    string
}

// DomainGraphics графика
type DomainGraphics struct {
	Type   string // vnc, spice
	Port   int
	AutoPort bool
	Listen string
}

// GenerateDomainXML генерирует XML для домена
func GenerateDomainXML(cfg *DomainConfig) string {
	var xml strings.Builder

	xml.WriteString("<domain")
	if cfg.UUID != "" {
		xml.WriteString(fmt.Sprintf(" type='kvm' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'"))
	} else {
		xml.WriteString(" type='kvm'")
	}
	xml.WriteString(">\n")

	// Имя
	xml.WriteString(fmt.Sprintf("  <name>%s</name>\n", cfg.Name))

	// UUID
	if cfg.UUID != "" {
		xml.WriteString(fmt.Sprintf("  <uuid>%s</uuid>\n", cfg.UUID))
	}

	// Метаданные для отслеживания
	xml.WriteString("  <metadata>\n")
	xml.WriteString("    <liblictl:lictl xmlns:liblictl='urn:lictl:metadata'>\n")
	xml.WriteString(fmt.Sprintf("      <liblictl:managed>true</liblictl:managed>\n"))
	xml.WriteString("    </liblictl:lictl>\n")
	xml.WriteString("  </metadata>\n")

	// Мemory
	xml.WriteString(fmt.Sprintf("  <memory unit='MiB'>%d</memory>\n", cfg.Memory))
	xml.WriteString(fmt.Sprintf("  <currentMemory unit='MiB'>%d</currentMemory>\n", cfg.Memory))

	// VCPU
	if cfg.VCPUPlacement == "" {
		cfg.VCPUPlacement = "static"
	}
	xml.WriteString(fmt.Sprintf("  <vcpu placement='%s'>%d</vcpu>\n", cfg.VCPUPlacement, cfg.VCPUs))

	// OS
	xml.WriteString("  <os>\n")
	arch := cfg.OSArch
	if arch == "" {
		arch = "x86_64"
	}
	machine := cfg.Machine
	if machine == "" {
		machine = "pc-q35-8.0"
	}
	xml.WriteString(fmt.Sprintf("    <type arch='%s' machine='%s'>hvm</type>\n", arch, machine))

	bootDev := cfg.BootDev
	if bootDev == "" {
		bootDev = "hd"
	}
	xml.WriteString(fmt.Sprintf("    <boot dev='%s'/>\n", bootDev))
	xml.WriteString("  </os>\n")

	// Features
	xml.WriteString("  <features>\n")
	xml.WriteString("    <acpi/>\n")
	xml.WriteString("    <apic/>\n")
	xml.WriteString("  </features>\n")

	// CPU mode
	xml.WriteString("  <cpu mode='host-passthrough' check='none'/>\n")

	// Clock
	xml.WriteString("  <clock offset='utc'>\n")
	xml.WriteString("    <timer name='rtc' tickpolicy='catchup'/>\n")
	xml.WriteString("    <timer name='pit' tickpolicy='delay'/>\n")
	xml.WriteString("    <timer name='hpet' present='no'/>\n")
	xml.WriteString("  </clock>\n")

	// Devices
	xml.WriteString("  <devices>\n")

	// Disks
	for i, disk := range cfg.Disks {
		xml.WriteString(fmt.Sprintf("    <disk type='%s' device='%s'>\n", disk.Type, disk.Device))
		xml.WriteString(fmt.Sprintf("      <driver name='qemu' type='%s'/>\n", disk.Format))

		if disk.Type == "file" {
			xml.WriteString(fmt.Sprintf("      <source file='%s'/>\n", disk.Source))
		} else if disk.Type == "volume" {
			// Формат: pool/volume
			parts := strings.SplitN(disk.Source, "/", 2)
			if len(parts) == 2 {
				xml.WriteString(fmt.Sprintf("      <source pool='%s' volume='%s'/>\n", parts[0], parts[1]))
			}
		}

		dev := disk.Target
		if dev == "" {
			if i == 0 {
				dev = "vda"
			} else {
				dev = fmt.Sprintf("vd%s", string(rune('a'+i)))
			}
		}
		xml.WriteString(fmt.Sprintf("      <target dev='%s' bus='%s'/>\n", dev, disk.Bus))

		if disk.ReadOnly {
			xml.WriteString("      <readonly/>\n")
		}
		xml.WriteString("    </disk>\n")
	}

	// Cloud-init ISO
	if cfg.CloudInitISO != "" {
		xml.WriteString("    <disk type='file' device='cdrom'>\n")
		xml.WriteString("      <driver name='qemu' type='raw'/>\n")
		xml.WriteString(fmt.Sprintf("      <source file='%s'/>\n", cfg.CloudInitISO))
		xml.WriteString("      <target dev='sda' bus='sata'/>\n")
		xml.WriteString("      <readonly/>\n")
		xml.WriteString("    </disk>\n")
	}

	// Interfaces
	for _, iface := range cfg.Interfaces {
		xml.WriteString(fmt.Sprintf("    <interface type='%s'>\n", iface.Type))

		if iface.Type == "network" {
			xml.WriteString(fmt.Sprintf("      <source network='%s'/>\n", iface.Source))
		} else if iface.Type == "bridge" {
			xml.WriteString(fmt.Sprintf("      <source bridge='%s'/>\n", iface.Source))
		}

		if iface.MAC != "" {
			xml.WriteString(fmt.Sprintf("      <mac address='%s'/>\n", iface.MAC))
		}

		model := iface.Model
		if model == "" {
			model = "virtio"
		}
		xml.WriteString(fmt.Sprintf("      <model type='%s'/>\n", model))
		xml.WriteString("    </interface>\n")
	}

	// Graphics
	if cfg.Graphics == nil {
		cfg.Graphics = &DomainGraphics{
			Type:     "vnc",
			Port:     -1,
			AutoPort: true,
		}
	}
	xml.WriteString(fmt.Sprintf("    <graphics type='%s' port='%d'", cfg.Graphics.Type, cfg.Graphics.Port))
	if cfg.Graphics.AutoPort {
		xml.WriteString(" autoport='yes'")
	}
	if cfg.Graphics.Listen != "" {
		xml.WriteString(fmt.Sprintf(" listen='%s'", cfg.Graphics.Listen))
	}
	xml.WriteString("/>\n")

	// Serial console
	xml.WriteString("    <serial type='pty'>\n")
	xml.WriteString("      <target port='0'/>\n")
	xml.WriteString("    </serial>\n")
	xml.WriteString("    <console type='pty'>\n")
	xml.WriteString("      <target type='serial' port='0'/>\n")
	xml.WriteString("    </console>\n")

	xml.WriteString("  </devices>\n")

	xml.WriteString("</domain>")
	return xml.String()
}
