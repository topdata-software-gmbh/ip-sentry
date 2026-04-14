package aggregator

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/topdata-software-gmbh/ip-sentry/internal/config"
	"github.com/topdata-software-gmbh/ip-sentry/internal/models"
)

type Detector struct {
	mu                  sync.Mutex
	counters            map[string][]time.Time
	lastBlock           map[string]time.Time
	threshold           int
	blacklistCountries  []string
	blacklistHosts      []string
	blacklistUserAgents []string
	whitelistHosts      []string
	whitelistIPRanges   []*net.IPNet
	whitelistIPs        []string
	blockCooldown       time.Duration
}

func NewDetector(cfg config.Config) *Detector {
	threshold := cfg.Thresholds.MaxRequestsPerMinute
	if threshold <= 0 {
		threshold = 100
	}

	return &Detector{
		counters:            make(map[string][]time.Time),
		lastBlock:           make(map[string]time.Time),
		threshold:           threshold,
		blacklistCountries:  cfg.Blacklist.Countries,
		blacklistHosts:      cfg.Blacklist.Hostnames,
		blacklistUserAgents: cfg.Blacklist.UserAgents,
		whitelistHosts:      cfg.Whitelist.Hostnames,
		whitelistIPs:        cfg.Whitelist.IPs,
		blockCooldown:       time.Minute,
	}
}

// SetWhitelistIPRanges replaces the set of whitelisted IP ranges (thread-safe).
func (d *Detector) SetWhitelistIPRanges(nets []*net.IPNet) {
	d.mu.Lock()
	d.whitelistIPRanges = nets
	d.mu.Unlock()
}

func (d *Detector) Process(entry *models.AccessLogEntry, country, hostname string) *models.BlockEvent {
	result := d.ProcessWithMetadata(entry, country, hostname)
	return result.Event
}

func (d *Detector) ProcessWithMetadata(entry *models.AccessLogEntry, country, hostname string) models.DetectionResult {
	if entry == nil || entry.IP == "" {
		return models.DetectionResult{}
	}

	now := entry.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isWhitelistedIPRange(entry.IP) {
		return models.DetectionResult{Mechanism: "WHITELIST_IP_RANGE", WhitelistIPRangeMatch: true}
	}

	if d.isWhitelistedHost(hostname) {
		return models.DetectionResult{Mechanism: "WHITELIST_HOSTNAME", WhitelistHostnameMatch: true}
	}

	if d.isWhitelistedIP(entry.IP) {
		return models.DetectionResult{Mechanism: "WHITELIST_IP", WhitelistIPMatch: true}
	}

	if d.isBlacklistedCountry(country) {
		return models.DetectionResult{
			Event:     d.makeBlockEvent(entry.IP, "BLACKLISTED_COUNTRY", country, hostname, now),
			Mechanism: "BLACKLISTED_COUNTRY",
		}
	}

	if d.isBlacklistedHost(hostname) {
		return models.DetectionResult{
			Event:     d.makeBlockEvent(entry.IP, "BLACKLISTED_HOSTNAME", country, hostname, now),
			Mechanism: "BLACKLISTED_HOSTNAME",
		}
	}

	if d.isBlacklistedUserAgent(entry.UserAgent) {
		return models.DetectionResult{
			Event:     d.makeBlockEvent(entry.IP, "BLACKLISTED_USER_AGENT", country, hostname, now),
			Mechanism: "BLACKLISTED_USER_AGENT",
		}
	}

	windowStart := now.Add(-1 * time.Minute)
	times := d.counters[entry.IP]
	filtered := times[:0]
	for _, t := range times {
		if t.After(windowStart) {
			filtered = append(filtered, t)
		}
	}
	filtered = append(filtered, now)
	d.counters[entry.IP] = filtered

	if len(filtered) > d.threshold {
		reason := fmt.Sprintf("RATE_LIMIT_EXCEEDED_%d_PER_MIN", d.threshold)
		return models.DetectionResult{
			Event:     d.makeBlockEvent(entry.IP, reason, country, hostname, now),
			Mechanism: "RATE_LIMIT_EXCEEDED",
		}
	}

	return models.DetectionResult{}
}

func (d *Detector) makeBlockEvent(ip, reason, country, hostname string, now time.Time) *models.BlockEvent {
	if last, ok := d.lastBlock[ip]; ok && now.Sub(last) < d.blockCooldown {
		return nil
	}
	d.lastBlock[ip] = now

	return &models.BlockEvent{
		IP:      ip,
		Reason:  reason,
		Country: country,
		Host:    hostname,
	}
}

func (d *Detector) isBlacklistedCountry(country string) bool {
	if country == "" {
		return false
	}
	for _, blocked := range d.blacklistCountries {
		if strings.EqualFold(strings.TrimSpace(blocked), strings.TrimSpace(country)) {
			return true
		}
	}
	return false
}

func (d *Detector) isBlacklistedHost(hostname string) bool {
	return matchesSuffixList(hostname, d.blacklistHosts)
}

func (d *Detector) isWhitelistedHost(hostname string) bool {
	return matchesSuffixList(hostname, d.whitelistHosts)
}

func (d *Detector) isWhitelistedIPRange(ipStr string) bool {
	if len(d.whitelistIPRanges) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, ipNet := range d.whitelistIPRanges {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func (d *Detector) isWhitelistedIP(ip string) bool {
	if ip == "" {
		return false
	}
	ip = strings.TrimSpace(ip)
	for _, whitelisted := range d.whitelistIPs {
		if strings.EqualFold(strings.TrimSpace(whitelisted), ip) {
			return true
		}
	}
	return false
}

func (d *Detector) isBlacklistedUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" || ua == "-" {
		return false
	}
	for _, blocked := range d.blacklistUserAgents {
		s := strings.ToLower(strings.TrimSpace(blocked))
		if s != "" && strings.Contains(ua, s) {
			return true
		}
	}
	return false
}

func matchesSuffixList(hostname string, suffixes []string) bool {
	host := strings.ToLower(strings.TrimSpace(hostname))
	if host == "" {
		return false
	}

	for _, suffix := range suffixes {
		s := strings.ToLower(strings.TrimSpace(suffix))
		if s != "" && strings.HasSuffix(host, s) {
			return true
		}
	}
	return false
}
