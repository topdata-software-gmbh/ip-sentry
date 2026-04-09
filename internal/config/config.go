package config

type Thresholds struct {
	MaxRequestsPerMinute int `mapstructure:"max_requests_per_minute"`
}

type Blacklist struct {
	Countries  []string `mapstructure:"countries"`
	Hostnames  []string `mapstructure:"hostnames"`
	UserAgents []string `mapstructure:"user_agents"`
}

type Whitelist struct {
	Hostnames []string `mapstructure:"hostnames"`
	IPs       []string `mapstructure:"ips"`
}

type Config struct {
	LogSources           []string   `mapstructure:"log_sources"`
	BlockLogOutput       string     `mapstructure:"block_log_output"`
	HeartbeatStatsOutput string     `mapstructure:"heartbeat_stats_output"`
	Fail2banConfigPath   string     `mapstructure:"fail2ban_config_path"`
	GeoIPDBPath          string     `mapstructure:"geoip_db_path"`
	Thresholds           Thresholds `mapstructure:"thresholds"`
	Blacklist            Blacklist  `mapstructure:"blacklist"`
	Whitelist            Whitelist  `mapstructure:"whitelist"`
}
