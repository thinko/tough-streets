// Package config provides the configuration structures for the server components
package config

import (
	appconfig "tough-streets/internal/config"
)

// Using type aliases to maintain backward compatibility
type DHCPConfig = appconfig.DHCPConfig
type DNSConfig = appconfig.DNSConfig
type ServerConfig = appconfig.ServerConfig
