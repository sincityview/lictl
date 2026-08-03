package xml

import (
	"fmt"
	"net"
	"strings"
)

// NetworkConfig конфигурация сети для генерации XML
type NetworkConfig struct {
	Name     string
	Bridge   string
	Mode     string // nat, route, isolated, bridge
	Subnet   string // CIDR: 10.0.0.0/24
	DHCP     *DHCPConfig
	DNS      *DNSConfig
	Autostart bool
}

// DHCPConfig конфигурация DHCP
type DHCPConfig struct {
	RangeStart string
	RangeEnd   string
	Hosts      []DHCPHost
}

// DHCPHost статическая DHCP-запись
type DHCPHost struct {
	MAC  string
	IP   string
	Name string
}

// DNSConfig конфигурация DNS
type DNSConfig struct {
	Enable    bool
	Forwarder string // IP DNS-форвардера
	Hosts     []DNSHost
}

// DNSHost статическая DNS-запись
type DNSHost struct {
	IP   string
	Hostnames []string
}

// GenerateNetworkXML генерирует XML для сети
func GenerateNetworkXML(cfg *NetworkConfig) string {
	var xml strings.Builder

	xml.WriteString("<network>\n")

	// Имя
	xml.WriteString(fmt.Sprintf("  <name>%s</name>\n", cfg.Name))

	// Bridge
	if cfg.Bridge != "" {
		xml.WriteString(fmt.Sprintf("  <bridge name='%s'/>\n", cfg.Bridge))
	}

	// MAC address
	xml.WriteString("  <mac address='52:54:00:00:00:01'/>\n")

	// Forward mode
	if cfg.Mode != "" && cfg.Mode != "isolated" {
		xml.WriteString(fmt.Sprintf("  <forward mode='%s'/>\n", cfg.Mode))
	}

	// IP configuration
	if cfg.Subnet != "" {
		ip, ipnet, err := net.ParseCIDR(cfg.Subnet)
		if err == nil {
			// Вычисляем gateway (первый IP в подсети)
			gateway := ip.Mask(ipnet.Mask)
			gateway[3]++ // Первый IP = gateway

			// Маска в формате libvirt
			var mask net.IPMask = ipnet.Mask
			xml.WriteString(fmt.Sprintf("  <ip address='%s' netmask='%s'>\n",
				gateway.String(), net.IP(mask).String()))

			// DHCP
			if cfg.DHCP != nil {
				xml.WriteString("    <dhcp>\n")

				if cfg.DHCP.RangeStart != "" && cfg.DHCP.RangeEnd != "" {
					xml.WriteString(fmt.Sprintf("      <range start='%s' end='%s'/>\n",
						cfg.DHCP.RangeStart, cfg.DHCP.RangeEnd))
				}

				for _, host := range cfg.DHCP.Hosts {
					xml.WriteString(fmt.Sprintf("      <host mac='%s' ip='%s' name='%s'/>\n",
						host.MAC, host.IP, host.Name))
				}

				xml.WriteString("    </dhcp>\n")
			}

			xml.WriteString("  </ip>\n")
		}
	}

	// DNS
	if cfg.DNS != nil && cfg.DNS.Enable {
		xml.WriteString("  <dns>\n")

		if cfg.DNS.Forwarder != "" {
			xml.WriteString(fmt.Sprintf("    <forwarder addr='%s'/>\n", cfg.DNS.Forwarder))
		}

		for _, host := range cfg.DNS.Hosts {
			for _, hostname := range host.Hostnames {
				xml.WriteString(fmt.Sprintf("    <host ip='%s'>\n", host.IP))
				xml.WriteString(fmt.Sprintf("      <hostname>%s</hostname>\n", hostname))
				xml.WriteString("    </host>\n")
			}
		}

		xml.WriteString("  </dns>\n")
	}

	xml.WriteString("</network>")
	return xml.String()
}

// ParseSubnet разбирает CIDR-подсеть
func ParseSubnet(cidr string) (gateway, netmask string, err error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}

	// Gateway - первый IP
	gw := ip.Mask(ipnet.Mask)
	gw[3]++

	// Netmask
	var mask net.IPMask = ipnet.Mask

	return gw.String(), net.IP(mask).String(), nil
}

// GenerateDHCPRange генерирует диапазон DHCP из подсети
func GenerateDHCPRange(cidr string, startOffset, endOffset int) (start, end string, err error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}

	// Начальный IP
	startIP := ip.Mask(ipnet.Mask)
	startIP[3] += byte(startOffset)

	// Конечный IP
	endIP := ip.Mask(ipnet.Mask)
	endIP[3] += byte(endOffset)

	return startIP.String(), endIP.String(), nil
}
