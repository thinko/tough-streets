// Package config provides application configuration structures
package config

import (
	appconfig "tough-streets/internal/config"
)

// Forward type definitions to the main config package
type DNSConfig = appconfig.DNSConfig
type DHCPConfig = appconfig.DHCPConfig
type ServerConfig = appconfig.ServerConfig
