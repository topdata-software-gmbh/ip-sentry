package monitor

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nxadm/tail"
	"github.com/oschwald/geoip2-golang"
	"github.com/topdata-software-gmbh/ip-sentry/internal/aggregator"
	"github.com/topdata-software-gmbh/ip-sentry/internal/config"
	"github.com/topdata-software-gmbh/ip-sentry/internal/models"
	"github.com/topdata-software-gmbh/ip-sentry/internal/parser"
	"github.com/topdata-software-gmbh/ip-sentry/internal/whitelist"
	"go.uber.org/zap"
)

type Monitor struct {
	cfg          config.Config
	detector     *aggregator.Detector
	logger       *zap.Logger
	geoDB        *geoip2.Reader
	blockLog     *os.File
	heartbeatLog *os.File
	hostCache    map[string]string
	hostCacheM   sync.RWMutex
	hostFlight   sync.Map
	stats        models.GlobalStats
}

func New(cfg config.Config) (*Monitor, error) {
	if len(cfg.LogSources) == 0 {
		return nil, fmt.Errorf("log_sources must contain at least one file")
	}
	if cfg.BlockLogOutput == "" {
		return nil, fmt.Errorf("block_log_output is required")
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	m := &Monitor{
		cfg:       cfg,
		detector:  aggregator.NewDetector(cfg),
		logger:    logger,
		hostCache: make(map[string]string),
	}

	m.logger.Info("Initializing monitor",
		zap.Int("sources", len(cfg.LogSources)),
		zap.String("output", cfg.BlockLogOutput),
	)

	heartbeatPath := cfg.HeartbeatStatsOutput
	if strings.TrimSpace(heartbeatPath) == "" {
		heartbeatPath = filepath.Join(filepath.Dir(cfg.BlockLogOutput), "ip-sentry-heartbeat-stats.log")
	}

	m.logger.Info("Checking infrastructure paths")
	for _, source := range cfg.LogSources {
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				m.logger.Warn("Log source not found", zap.String("path", source))
				continue
			}
			m.logger.Warn("Log source check failed", zap.String("path", source), zap.Error(err))
			continue
		}
		m.logger.Info("Log source verified", zap.String("path", source))
	}

	if cfg.Fail2banConfigPath != "" {
		if fi, err := os.Stat(cfg.Fail2banConfigPath); err != nil {
			if os.IsNotExist(err) {
				m.logger.Warn("Fail2ban config path not found", zap.String("path", cfg.Fail2banConfigPath))
			} else {
				m.logger.Warn("Fail2ban config path check failed", zap.String("path", cfg.Fail2banConfigPath), zap.Error(err))
			}
		} else if !fi.IsDir() {
			m.logger.Warn("Fail2ban config path is not a directory", zap.String("path", cfg.Fail2banConfigPath))
		} else {
			m.logger.Info("Fail2ban config path verified", zap.String("path", cfg.Fail2banConfigPath))
		}
	}

	if err := os.MkdirAll(filepath.Dir(cfg.BlockLogOutput), 0o755); err != nil {
		return nil, fmt.Errorf("create block log directory: %w", err)
	}

	m.logger.Info("Opening bridge log", zap.String("path", cfg.BlockLogOutput))
	blockLog, err := os.OpenFile(cfg.BlockLogOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open block log output: %w", err)
	}
	m.blockLog = blockLog

	if err := os.MkdirAll(filepath.Dir(heartbeatPath), 0o755); err != nil {
		return nil, fmt.Errorf("create heartbeat stats directory: %w", err)
	}

	m.logger.Info("Opening heartbeat stats log", zap.String("path", heartbeatPath))
	heartbeatLog, err := os.OpenFile(heartbeatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open heartbeat stats output: %w", err)
	}
	m.heartbeatLog = heartbeatLog

	if cfg.GeoIPDBPath != "" {
		if _, err := os.Stat(cfg.GeoIPDBPath); err == nil {
			geoDB, err := geoip2.Open(cfg.GeoIPDBPath)
			if err != nil {
				m.logger.Warn("failed to open GeoIP DB", zap.String("path", cfg.GeoIPDBPath), zap.Error(err))
			} else {
				m.geoDB = geoDB
				m.logger.Info("GeoIP database loaded")
			}
		} else {
			m.logger.Warn("GeoIP DB missing; country checks disabled", zap.String("path", cfg.GeoIPDBPath))
		}
	}

	// --- Whitelist IP ranges ---
	if err := m.loadWhitelistIPRanges(); err != nil {
		m.logger.Warn("Failed to load whitelist IP ranges", zap.Error(err))
	}

	return m, nil
}

func (m *Monitor) Close() {
	if m.geoDB != nil {
		_ = m.geoDB.Close()
	}
	if m.blockLog != nil {
		_ = m.blockLog.Close()
	}
	if m.heartbeatLog != nil {
		_ = m.heartbeatLog.Close()
	}
	_ = m.logger.Sync()
}

func (m *Monitor) Run(ctx context.Context) error {
	lineCh := make(chan string, 1024)
	errCh := make(chan error, len(m.cfg.LogSources))
	var wg sync.WaitGroup

	m.logger.Info("Monitor engine starting")
	go m.startHeartbeat(ctx)
	m.logger.Info("Tailing log files", zap.Int("count", len(m.cfg.LogSources)))
	for _, source := range m.cfg.LogSources {
		source := source
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.logger.Info("Starting tailer for source", zap.String("path", source))
			m.tailSource(ctx, source, lineCh, errCh)
		}()
	}

	go func() {
		wg.Wait()
		close(lineCh)
	}()

	var workerWg sync.WaitGroup
	for i := 0; i < 100; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case line, ok := <-lineCh:
					if !ok {
						return
					}
					m.stats.IncrementProcessed()
					if err := m.processLine(line); err != nil {
						m.logger.Warn("line processing failed", zap.Error(err))
					}
				}
			}
		}()
	}

	doneCh := make(chan struct{})
	go func() {
		workerWg.Wait()
		close(doneCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-doneCh:
			return nil
		}
	}
}

func formatTopItems(items []models.TopItem) string {
	if len(items) == 0 {
		return "none"
	}
	var parts []string
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s:%d", it.Key, it.Count))
	}
	return strings.Join(parts, ", ")
}

func (m *Monitor) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed := m.stats.Processed()
			parsed := m.stats.Parsed()
			blocks := m.stats.Blocks()
			whitelistHits := m.stats.WhitelistHostnameHits()
			whitelistIPRangeHits := m.stats.WhitelistIPRangeHits()
			blockedByCountry := m.stats.BlockedByCountry()
			blockedByHostname := m.stats.BlockedByHostname()
			blockedByUserAgent := m.stats.BlockedByUserAgent()
			blockedByRateLimit := m.stats.BlockedByRateLimit()
			blockedByOther := m.stats.BlockedByOther()

			m.logger.Info("HEARTBEAT STATS",
				zap.Uint64("processed_lines", processed),
				zap.Uint64("successfully_parsed", parsed),
				zap.Uint64("blocks_generated", blocks),
				zap.Uint64("hostname_whitelist_hits", whitelistHits),
				zap.Uint64("ip_range_whitelist_hits", whitelistIPRangeHits),
				zap.Uint64("blocked_by_country", blockedByCountry),
				zap.Uint64("blocked_by_hostname", blockedByHostname),
				zap.Uint64("blocked_by_user_agent", blockedByUserAgent),
				zap.Uint64("blocked_by_rate_limit", blockedByRateLimit),
				zap.Uint64("blocked_by_other", blockedByOther),
			)

			if m.heartbeatLog != nil {
				_, _ = fmt.Fprintf(
					m.heartbeatLog,
					"%s HEARTBEAT_DETAIL processed_lines=%d successfully_parsed=%d blocks_generated=%d hostname_whitelist_hits=%d ip_range_whitelist_hits=%d blocked_by_country=%d blocked_by_hostname=%d blocked_by_user_agent=%d blocked_by_rate_limit=%d blocked_by_other=%d top_countries=%q top_user_agents=%q\n",
					time.Now().UTC().Format(time.RFC3339),
					processed,
					parsed,
					blocks,
					whitelistHits,
					whitelistIPRangeHits,
					blockedByCountry,
					blockedByHostname,
					blockedByUserAgent,
					blockedByRateLimit,
					blockedByOther,
					formatTopItems(m.stats.TopCountries(5)),
					formatTopItems(m.stats.TopUserAgents(5)),
				)
			}
		}
	}
}

func (m *Monitor) tailSource(ctx context.Context, source string, lineCh chan<- string, errCh chan<- error) {
	t, err := tail.TailFile(source, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      true,
	})
	if err != nil {
		errCh <- fmt.Errorf("tail %s: %w", source, err)
		return
	}
	defer func() { _ = t.Stop() }()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-t.Lines:
			if !ok {
				// Channel closed (e.g., empty file) - wait for context instead of exiting
				<-ctx.Done()
				return
			}
			if line == nil {
				continue
			}
			if line.Err != nil {
				errCh <- fmt.Errorf("tail line error (%s): %w", source, line.Err)
				continue
			}
			if strings.TrimSpace(line.Text) == "" {
				continue
			}
			select {
			case lineCh <- line.Text:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (m *Monitor) processLine(line string) error {
	entry := parser.ParseLine(line)
	if entry == nil {
		return nil
	}
	m.stats.IncrementParsed()

	country := m.lookupCountry(entry.IP)
	hostname := m.lookupHostname(entry.IP)

	m.stats.RecordRequest(country, entry.UserAgent)

	result := m.detector.ProcessWithMetadata(entry, country, hostname)
	if result.WhitelistHostnameMatch {
		m.stats.IncrementWhitelistHostnameHits()
	}
	if result.WhitelistIPRangeMatch {
		m.stats.IncrementWhitelistIPRangeHits()
	}

	event := result.Event
	if event == nil {
		return nil
	}
	m.stats.IncrementBlocks()
	m.stats.IncrementBlocksByMechanism(result.Mechanism)

	_, err := fmt.Fprintf(
		m.blockLog,
		"%s - BLOCK_REQUESTED - Reason:%s Country:%s Host:%s\n",
		event.IP,
		normalizeField(event.Reason),
		normalizeField(event.Country),
		normalizeField(event.Host),
	)
	if err != nil {
		return fmt.Errorf("write block event: %w", err)
	}

	m.logger.Info(
		"block requested",
		zap.String("ip", event.IP),
		zap.String("reason", event.Reason),
		zap.String("country", event.Country),
		zap.String("host", event.Host),
	)

	return nil
}

func (m *Monitor) lookupCountry(ipStr string) string {
	if m.geoDB == nil {
		return ""
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	rec, err := m.geoDB.City(ip)
	if err != nil || rec == nil || rec.Country.IsoCode == "" {
		return ""
	}
	return rec.Country.IsoCode
}

func (m *Monitor) lookupHostname(ip string) string {
	m.hostCacheM.RLock()
	if host, ok := m.hostCache[ip]; ok {
		m.hostCacheM.RUnlock()
		return host
	}
	m.hostCacheM.RUnlock()

	chI, loaded := m.hostFlight.LoadOrStore(ip, make(chan struct{}))
	if loaded {
		<-chI.(chan struct{})
		m.hostCacheM.RLock()
		host := m.hostCache[ip]
		m.hostCacheM.RUnlock()
		return host
	}
	defer func() {
		close(chI.(chan struct{}))
		m.hostFlight.Delete(ip)
	}()

	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		m.hostCacheM.Lock()
		m.hostCache[ip] = ""
		m.hostCacheM.Unlock()
		return ""
	}

	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(names[0])), ".")
	m.hostCacheM.Lock()
	m.hostCache[ip] = host
	m.hostCacheM.Unlock()
	return host
}

func (m *Monitor) loadWhitelistIPRanges() error {
	var allNets []*net.IPNet

	// Parse static IP ranges from config
	if len(m.cfg.Whitelist.IPRanges) > 0 {
		nets, err := whitelist.ParseStaticRanges(m.cfg.Whitelist.IPRanges)
		if err != nil {
			return fmt.Errorf("parse static IP ranges: %w", err)
		}
		m.logger.Info("Loaded static whitelist IP ranges", zap.Int("count", len(nets)))
		allNets = append(allNets, nets...)
	}

	// Fetch Cloudflare IP ranges
	if m.cfg.Whitelist.Cloudflare.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		nets, err := whitelist.FetchCloudflareRanges(ctx, m.logger)
		if err != nil {
			return fmt.Errorf("fetch Cloudflare IP ranges: %w", err)
		}
		allNets = append(allNets, nets...)
	}

	if len(allNets) > 0 {
		m.detector.SetWhitelistIPRanges(allNets)
		m.logger.Info("Whitelist IP ranges active", zap.Int("total_ranges", len(allNets)))
	}

	return nil
}

func normalizeField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
