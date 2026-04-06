package config

type Thresholds struct {
	MaxRequestsPerMinute int `mapstructure:"max_requests_per_minute"`
}

type Blacklist struct {
	Countries []string `mapstructure:"countries"`
	Hostnames []string `mapstructure:"hostnames"`
}

type Whitelist struct {
	Hostnames []string `mapstructure:"hostnames"`
}

type Config struct {
	LogSources     []string   `mapstructure:"log_sources"`
	BlockLogOutput string     `mapstructure:"block_log_output"`
	GeoIPDBPath    string     `mapstructure:"geoip_db_path"`
	Thresholds     Thresholds `mapstructure:"thresholds"`
	Blacklist      Blacklist  `mapstructure:"blacklist"`
	Whitelist      Whitelist  `mapstructure:"whitelist"`
}
