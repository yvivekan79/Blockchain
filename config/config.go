package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Node      NodeConfig      `mapstructure:"node"`
	Server    ServerConfig    `mapstructure:"server"`
	Consensus ConsensusConfig `mapstructure:"consensus"`
	Sharding  ShardingConfig  `mapstructure:"sharding"`
	Network   NetworkConfig   `mapstructure:"network"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Security  SecurityConfig  `mapstructure:"security"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

type BootstrapConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	AdvertiseAddress string `mapstructure:"advertise_address"`
}

type NodeConfig struct {
	ID                 string `mapstructure:"id"`
	Name               string `mapstructure:"name"`
	Description        string `mapstructure:"description"`
	ConsensusAlgorithm string `mapstructure:"consensus_algorithm"`
	Role               string `mapstructure:"role"`
	ExternalIP         string `mapstructure:"external_ip"`
	Region             string `mapstructure:"region"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

type ConsensusConfig struct {
	Algorithm    string  `mapstructure:"algorithm"`
	Difficulty   int     `mapstructure:"difficulty"`
	BlockTime    int     `mapstructure:"block_time"`
	MinStake     int64   `mapstructure:"min_stake"`
	StakeRatio   float64 `mapstructure:"stake_ratio"`
	ViewTimeout  int     `mapstructure:"view_timeout"`
	Byzantine    int     `mapstructure:"byzantine"`
	LayerDepth   int     `mapstructure:"layer_depth"`
	ChannelCount int     `mapstructure:"channel_count"`
	GasLimit     int64   `mapstructure:"gas_limit"`
}

type ShardingConfig struct {
	NumShards        int     `mapstructure:"num_shards"`
	ShardSize        int     `mapstructure:"shard_size"`
	CrossShardDelay  int     `mapstructure:"cross_shard_delay"`
	RebalanceThresh  float64 `mapstructure:"rebalance_threshold"`
	LayeredStructure bool    `mapstructure:"layered_structure"`
}

type NetworkConfig struct {
	Port         int      `mapstructure:"port"`
	MaxPeers     int      `mapstructure:"max_peers"`
	Seeds        []string `mapstructure:"seeds"`
	BootNodes    []string `mapstructure:"boot_nodes"`
	Timeout      int      `mapstructure:"timeout"`
	KeepAlive    int      `mapstructure:"keep_alive"`
	ExternalIP   string   `mapstructure:"external_ip"`
	BindAddress  string   `mapstructure:"bind_address"`
	Encryption   bool     `mapstructure:"encryption"`
	AuthRequired bool     `mapstructure:"auth_required"`
}

type StorageConfig struct {
	DataDir    string `mapstructure:"data_dir"`
	CacheSize  int    `mapstructure:"cache_size"`
	Compact    bool   `mapstructure:"compact"`
	Encryption bool   `mapstructure:"encryption"`
}

type SecurityConfig struct {
	JWTSecret       string `mapstructure:"jwt_secret"`
	TLSEnabled      bool   `mapstructure:"tls_enabled"`
	CertFile        string `mapstructure:"cert_file"`
	KeyFile         string `mapstructure:"key_file"`
	RateLimit       int    `mapstructure:"rate_limit"`
	MaxConnections  int    `mapstructure:"max_connections"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/lscc/")

	// Set default values
	setDefaults()

	// Enable reading from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("LSCC")

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Override with environment variables if present
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		if port, err := strconv.Atoi(envPort); err == nil {
			viper.Set("server.port", port)
		}
	}
	if envAlgorithm := os.Getenv("CONSENSUS_ALGORITHM"); envAlgorithm != "" {
		viper.Set("consensus.algorithm", envAlgorithm)
		viper.Set("node.consensus_algorithm", envAlgorithm)
	}
	if envP2PPort := os.Getenv("P2P_PORT"); envP2PPort != "" {
		if port, err := strconv.Atoi(envP2PPort); err == nil {
			viper.Set("network.port", port)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override with environment variables
	overrideWithEnv(&config)

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// LoadConfigFromPath loads configuration from a specific file path
func LoadConfigFromPath(configPath string) (*Config, error) {
	viper.Reset() // Reset any previous configuration

	// Set the config file explicitly
	viper.SetConfigFile(configPath)

	// Set default values
	setDefaults()

	// Enable reading from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("LSCC")

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file %s: %w", configPath, err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override with environment variables
	overrideWithEnv(&config)

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "LSCC Blockchain")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.environment", "development")

	// Node defaults
	viper.SetDefault("node.id", generateNodeID())
	viper.SetDefault("node.name", "LSCC-Node")
	viper.SetDefault("node.description", "LSCC Blockchain Node")
	viper.SetDefault("node.consensus_algorithm", "lscc")
	viper.SetDefault("node.role", "validator")
	viper.SetDefault("node.external_ip", "")
	viper.SetDefault("node.region", "local")

	// Server defaults
	viper.SetDefault("server.port", 5000)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "development")

	// Consensus defaults
	viper.SetDefault("consensus.algorithm", "lscc")
	viper.SetDefault("consensus.difficulty", 4)
	viper.SetDefault("consensus.block_time", 10)
	viper.SetDefault("consensus.min_stake", 1000)
	viper.SetDefault("consensus.stake_ratio", 0.1)
	viper.SetDefault("consensus.view_timeout", 30)
	viper.SetDefault("consensus.byzantine", 1)
	viper.SetDefault("consensus.layer_depth", 3)
	viper.SetDefault("consensus.channel_count", 5)

	// Sharding defaults
	viper.SetDefault("sharding.num_shards", 4)
	viper.SetDefault("sharding.shard_size", 100)
	viper.SetDefault("sharding.cross_shard_delay", 100)
	viper.SetDefault("sharding.rebalance_threshold", 0.7)
	viper.SetDefault("sharding.layered_structure", true)

	// Network defaults
	viper.SetDefault("network.port", 9000)
	viper.SetDefault("network.max_peers", 50)
	viper.SetDefault("network.timeout", 30)
	viper.SetDefault("network.keep_alive", 60)
	viper.SetDefault("network.external_ip", "")
	viper.SetDefault("network.bind_address", "0.0.0.0")
	viper.SetDefault("network.encryption", false)
	viper.SetDefault("network.auth_required", false)

	// Bootstrap defaults
	viper.SetDefault("bootstrap.enabled", false)
	viper.SetDefault("bootstrap.advertise_address", "")

	// Storage defaults
	viper.SetDefault("storage.data_dir", "./data")
	viper.SetDefault("storage.cache_size", 100)
	viper.SetDefault("storage.compact", true)
	viper.SetDefault("storage.encryption", false)

	// Security defaults
	viper.SetDefault("security.jwt_secret", "default-jwt-secret-change-in-production")
	viper.SetDefault("security.tls_enabled", false)
	viper.SetDefault("security.rate_limit", 100)
	viper.SetDefault("security.max_connections", 1000)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("logging.compress", true)
}

func overrideWithEnv(config *Config) {
	// Override sensitive values with environment variables
	if secret := os.Getenv("LSCC_JWT_SECRET"); secret != "" {
		config.Security.JWTSecret = secret
	}

	if dataDir := os.Getenv("LSCC_DATA_DIR"); dataDir != "" {
		config.Storage.DataDir = dataDir
	}

	if certFile := os.Getenv("LSCC_CERT_FILE"); certFile != "" {
		config.Security.CertFile = certFile
	}

	if keyFile := os.Getenv("LSCC_KEY_FILE"); keyFile != "" {
		config.Security.KeyFile = keyFile
	}
}

func validateConfig(config *Config) error {
	// Validate consensus algorithm
	validConsensus := map[string]bool{
		"pow": true, "pos": true, "pbft": true, "ppbft": true, "lscc": true,
	}
	if !validConsensus[config.Consensus.Algorithm] {
		return fmt.Errorf("invalid consensus algorithm: %s", config.Consensus.Algorithm)
	}

	// Validate ports
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Network.Port < 1 || config.Network.Port > 65535 {
		return fmt.Errorf("invalid network port: %d", config.Network.Port)
	}

	// Validate sharding configuration
	if config.Sharding.NumShards < 1 {
		return fmt.Errorf("number of shards must be at least 1")
	}

	if config.Sharding.ShardSize < 1 {
		return fmt.Errorf("shard size must be at least 1")
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.Storage.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	return nil
}

func generateNodeID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("lscc-node-%s-%d", hostname, os.Getpid())
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	if path := os.Getenv("LSCC_CONFIG_PATH"); path != "" {
		return path
	}

	// Check common locations
	locations := []string{
		"./config/config.yaml",
		"./config.yaml",
		"/etc/lscc/config.yaml",
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			abs, _ := filepath.Abs(location)
			return abs
		}
	}

	return "./config.yaml"
}