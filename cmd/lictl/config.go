package main

import "github.com/alex/lictl/internal/config"

// Типы импортируются из internal/config
type Config = config.Config
type Provider = config.Provider
type LibvirtProvider = config.LibvirtProvider
type Resources = config.Resources
type StorageConfig = config.StorageConfig
type NetworkConfig = config.NetworkConfig
type DHCPConfig = config.DHCPConfig
type DNSConfig = config.DNSConfig
type VMConfig = config.VMConfig
type VMNetwork = config.VMNetwork
type CloudInit = config.CloudInit
type CIUser = config.CIUser
