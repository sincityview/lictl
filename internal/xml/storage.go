package xml

import (
	"fmt"
	"strings"
)

// StoragePoolConfig конфигурация пула хранения
type StoragePoolConfig struct {
	Name     string
	Type     string // dir, logical, fs, netfs, iscsi, etc.
	Path     string // для type=dir
	VGName   string // для type=logical
	TargetPath string // целевая директория
	Format   string // для volume
	Autostart bool
}

// StorageVolumeConfig конфигурация тома
type StorageVolumeConfig struct {
	Name     string
	Pool     string
	Capacity int64  // байты
	Format   string // qcow2, raw, etc.
	Path     string // путь к файлу (для dir pool)
	BackingStore string // backing store для снапшотов
}

// GenerateStoragePoolXML генерирует XML для пула хранения
func GenerateStoragePoolXML(cfg *StoragePoolConfig) string {
	var xml strings.Builder

	xml.WriteString("<pool type='")
	xml.WriteString(cfg.Type)
	xml.WriteString("'>\n")

	xml.WriteString(fmt.Sprintf("  <name>%s</name>\n", cfg.Name))

	switch cfg.Type {
	case "dir":
		xml.WriteString("  <target>\n")
		xml.WriteString(fmt.Sprintf("    <path>%s</path>\n", cfg.Path))
		xml.WriteString("  </target>\n")

	case "logical":
		xml.WriteString(fmt.Sprintf("  <source>\n"))
		xml.WriteString(fmt.Sprintf("    <name>%s</name>\n", cfg.VGName))
		xml.WriteString("  </source>\n")
		if cfg.TargetPath != "" {
			xml.WriteString("  <target>\n")
			xml.WriteString(fmt.Sprintf("    <path>%s</path>\n", cfg.TargetPath))
			xml.WriteString("  </target>\n")
		}

	case "fs", "netfs":
		if cfg.TargetPath != "" {
			xml.WriteString("  <target>\n")
			xml.WriteString(fmt.Sprintf("    <path>%s</path>\n", cfg.TargetPath))
			xml.WriteString("  </target>\n")
		}

	default:
		// Для остальных типов — базовая структура
		if cfg.Path != "" {
			xml.WriteString("  <target>\n")
			xml.WriteString(fmt.Sprintf("    <path>%s</path>\n", cfg.Path))
			xml.WriteString("  </target>\n")
		}
	}

	xml.WriteString("</pool>")
	return xml.String()
}

// GenerateStorageVolumeXML генерирует XML для тома
func GenerateStorageVolumeXML(cfg *StorageVolumeConfig) string {
	var xml strings.Builder

	xml.WriteString("<volume>\n")

	xml.WriteString(fmt.Sprintf("  <name>%s</name>\n", cfg.Name))

	// Capacity
	if cfg.Capacity > 0 {
		xml.WriteString(fmt.Sprintf("  <capacity unit='bytes'>%d</capacity>\n", cfg.Capacity))
	}

	// Allocation (0 = thin provisioning)
	xml.WriteString("  <allocation unit='bytes'>0</allocation>\n")

	// Target
	xml.WriteString("  <target>\n")
	if cfg.Path != "" {
		xml.WriteString(fmt.Sprintf("    <path>%s</path>\n", cfg.Path))
	}
	xml.WriteString(fmt.Sprintf("    <format type='%s'/>\n", cfg.Format))
	xml.WriteString("  </target>\n")

	// Backing store для снапшотов
	if cfg.BackingStore != "" {
		xml.WriteString("  <backingStore>\n")
		xml.WriteString(fmt.Sprintf("    <path>%s</path>\n", cfg.BackingStore))
		xml.WriteString(fmt.Sprintf("    <format type='%s'/>\n", cfg.Format))
		xml.WriteString("  </backingStore>\n")
	}

	xml.WriteString("</volume>")
	return xml.String()
}

// ParseDiskSize парсит размер диска (например "20Gi", "100G", "512M")
func ParseDiskSize(size string) (int64, error) {
	var multiplier int64 = 1
	var numStr strings.Builder

	for _, c := range size {
		if c >= '0' && c <= '9' || c == '.' {
			numStr.WriteRune(c)
		} else {
			switch strings.ToUpper(string(c)) {
			case "K", "KB":
				multiplier = 1024
			case "M", "MB":
				multiplier = 1024 * 1024
			case "G", "GB":
				multiplier = 1024 * 1024 * 1024
			case "T", "TB":
				multiplier = 1024 * 1024 * 1024 * 1024
			}
			break
		}
	}

	var num float64
	_, err := fmt.Sscanf(numStr.String(), "%f", &num)
	if err != nil {
		return 0, fmt.Errorf("невалидный размер: %s", size)
	}

	return int64(num * float64(multiplier)), nil
}

// FormatDiskSize форматирует размер в человекочитаемый вид
func FormatDiskSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
