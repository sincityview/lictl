package config

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var rangePattern = regexp.MustCompile(`\{(\d+)\.\.(\d+)\}`)

// ExpandRanges расширяет диапазоны в именах ресурсов
// Например: "worker-{1..3}" → ["worker-1", "worker-2", "worker-3"]
func ExpandRanges(name string) []string {
	matches := rangePattern.FindStringSubmatch(name)
	if matches == nil {
		return []string{name}
	}

	start, _ := strconv.Atoi(matches[1])
	end, _ := strconv.Atoi(matches[2])

	if start > end {
		start, end = end, start
	}

	result := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		expanded := strings.Replace(name, matches[0], strconv.Itoa(i), 1)
		result = append(result, expanded)
	}

	return result
}

// ExpandVMConfig расширяет VM-конфигурацию с диапазонами имён
// Возвращает список VM-конфигов, по одному на каждое расширенное имя
func ExpandVMConfig(vm VMConfig) []VMConfig {
	names := ExpandRanges(vm.Name)
	if len(names) == 1 {
		return []VMConfig{vm}
	}

	result := make([]VMConfig, 0, len(names))
	for i, name := range names {
		expanded := vm
		expanded.Name = name

		// Извлекаем числовой суффикс из имени
		numPart := ""
		if idx := strings.LastIndex(name, "-"); idx != -1 {
			numPart = name[idx+1:]
		}

		// Расширяем hostname и IP в cloud-init
		if vm.CloudInit != nil {
			ci := *vm.CloudInit // deep copy

			if strings.Contains(ci.Hostname, "{N}") {
				ci.Hostname = strings.Replace(ci.Hostname, "{N}", numPart, 1)
			}

			// Если ip_address_start указан — вычисляем IP для каждой VM
			if ci.Network != nil && ci.Network.IPStart != "" {
				netCopy := *ci.Network
				prefix := netCopy.IPPrefix
				if prefix == 0 {
					prefix = 24
				}
				ip := incrementIP(netCopy.IPStart, i)
				netCopy.IP = ip
				netCopy.IPPrefix = prefix
				ci.Network = &netCopy
			}

			expanded.CloudInit = &ci
		}

		result = append(result, expanded)
	}

	return result
}

// incrementIP увеличивает последний октет IP на n
func incrementIP(ipStr string, n int) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ipStr
	}
	ip4[3] += byte(n)
	return ip4.String()
}

// ExpandAllVMs расширяет все VM-конфигурации с диапазонами
func ExpandAllVMs(vms []VMConfig) []VMConfig {
	result := make([]VMConfig, 0, len(vms))
	for _, vm := range vms {
		result = append(result, ExpandVMConfig(vm)...)
	}
	return result
}

// ValidateSubnets проверяет что подсети не пересекаются
func ValidateSubnets(networks []NetworkConfig) error {
	subnets := make(map[string]string) // subnet → network name

	for _, n := range networks {
		if existing, ok := subnets[n.Subnet]; ok {
			return fmt.Errorf("сеть %s: подсеть %s уже используется сетью %s",
				n.Name, n.Subnet, existing)
		}
		subnets[n.Subnet] = n.Name
	}

	return nil
}

// GetNetworkNames возвращает отсортированный список имён сетей
func GetNetworkNames(networks []NetworkConfig) []string {
	names := make([]string, 0, len(networks))
	for _, n := range networks {
		names = append(names, n.Name)
	}
	sort.Strings(names)
	return names
}

// GetStorageNames возвращает отсортированный список имён пулов
func GetStorageNames(storage []StorageConfig) []string {
	names := make([]string, 0, len(storage))
	for _, s := range storage {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}
