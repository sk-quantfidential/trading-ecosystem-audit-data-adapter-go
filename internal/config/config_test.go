package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveSchemaName(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		instanceName string
		expected     string
	}{
		{
			name:         "singleton audit-correlator",
			serviceName:  "audit-correlator",
			instanceName: "audit-correlator",
			expected:     "audit",
		},
		{
			name:         "multi-instance exchange-OKX",
			serviceName:  "exchange-simulator",
			instanceName: "exchange-OKX",
			expected:     "exchange_okx",
		},
		{
			name:         "multi-instance custodian-Komainu",
			serviceName:  "custodian-simulator",
			instanceName: "custodian-Komainu",
			expected:     "custodian_komainu",
		},
		{
			name:         "multi-instance market-data-Coinmetrics",
			serviceName:  "market-data-simulator",
			instanceName: "market-data-Coinmetrics",
			expected:     "market_data_coinmetrics",
		},
		{
			name:         "singleton test-coordinator",
			serviceName:  "test-coordinator",
			instanceName: "test-coordinator",
			expected:     "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveSchemaName(tt.serviceName, tt.instanceName)
			assert.Equal(t, tt.expected, result,
				"Schema derivation failed for %s (service=%s, instance=%s)",
				tt.name, tt.serviceName, tt.instanceName)
		})
	}
}

func TestDeriveRedisNamespace(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		instanceName string
		expected     string
	}{
		{
			name:         "singleton audit-correlator",
			serviceName:  "audit-correlator",
			instanceName: "audit-correlator",
			expected:     "audit",
		},
		{
			name:         "multi-instance exchange-OKX",
			serviceName:  "exchange-simulator",
			instanceName: "exchange-OKX",
			expected:     "exchange:OKX",
		},
		{
			name:         "multi-instance custodian-Komainu",
			serviceName:  "custodian-simulator",
			instanceName: "custodian-Komainu",
			expected:     "custodian:Komainu",
		},
		{
			name:         "multi-instance exchange-Binance",
			serviceName:  "exchange-simulator",
			instanceName: "exchange-Binance",
			expected:     "exchange:Binance",
		},
		{
			name:         "singleton test-coordinator",
			serviceName:  "test-coordinator",
			instanceName: "test-coordinator",
			expected:     "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveRedisNamespace(tt.serviceName, tt.instanceName)
			assert.Equal(t, tt.expected, result,
				"Redis namespace derivation failed for %s (service=%s, instance=%s)",
				tt.name, tt.serviceName, tt.instanceName)
		})
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PostgreSQL URL with password",
			input:    "postgres://user:secretpass@localhost:5432/db",
			expected: "postgres://user:***@localhost:5432/db",
		},
		{
			name:     "Redis URL with password",
			input:    "redis://default:mypassword@localhost:6379/0",
			expected: "redis://default:***@localhost:6379/0",
		},
		{
			name:     "URL without password",
			input:    "postgres://localhost:5432/db",
			expected: "postgres://localhost:5432/db",
		},
		{
			name:     "MongoDB URL with password",
			input:    "mongodb://admin:admin123@localhost:27017/audit",
			expected: "mongodb://admin:***@localhost:27017/audit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPassword(tt.input)
			assert.Equal(t, tt.expected, result,
				"Password masking failed for %s", tt.name)
			assert.NotContains(t, result, "secretpass", "Password still visible in masked URL")
			assert.NotContains(t, result, "mypassword", "Password still visible in masked URL")
			assert.NotContains(t, result, "admin123", "Password still visible in masked URL")
		})
	}
}

func TestLoadRepositoryConfig_DefaultValues(t *testing.T) {
	// Clear environment variables to test defaults
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("SERVICE_INSTANCE_NAME", "")

	cfg := LoadRepositoryConfig()

	// Test defaults
	assert.Equal(t, "audit-correlator", cfg.ServiceName, "Default service name should be audit-correlator")
	assert.Equal(t, "audit-correlator", cfg.ServiceInstanceName, "Default instance name should match service name")
	assert.Equal(t, "audit", cfg.SchemaName, "Default schema should be derived from service name")
	assert.Equal(t, "audit", cfg.RedisNamespace, "Default namespace should be derived from service name")
	assert.Equal(t, "development", cfg.Environment, "Default environment should be development")
}

func TestLoadRepositoryConfig_CustomInstance(t *testing.T) {
	// Set environment variables for multi-instance
	t.Setenv("SERVICE_NAME", "exchange-simulator")
	t.Setenv("SERVICE_INSTANCE_NAME", "exchange-OKX")

	cfg := LoadRepositoryConfig()

	// Test multi-instance derivation
	assert.Equal(t, "exchange-simulator", cfg.ServiceName)
	assert.Equal(t, "exchange-OKX", cfg.ServiceInstanceName)
	assert.Equal(t, "exchange_okx", cfg.SchemaName, "Schema should be derived from instance name")
	assert.Equal(t, "exchange:OKX", cfg.RedisNamespace, "Namespace should be derived from instance name")
}

func TestLoadRepositoryConfig_BackwardCompatibility(t *testing.T) {
	// Only set SERVICE_NAME (old behavior)
	t.Setenv("SERVICE_NAME", "audit-correlator")
	t.Setenv("SERVICE_INSTANCE_NAME", "") // Not set

	cfg := LoadRepositoryConfig()

	// Test backward compatibility: instance name should default to service name
	assert.Equal(t, "audit-correlator", cfg.ServiceName)
	assert.Equal(t, "audit-correlator", cfg.ServiceInstanceName, "Instance name should default to service name for backward compatibility")
	assert.Equal(t, "audit", cfg.SchemaName)
	assert.Equal(t, "audit", cfg.RedisNamespace)
}
