package aggregator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/topdata-software-gmbh/ip-sentry/internal/config"
	"github.com/topdata-software-gmbh/ip-sentry/internal/models"
)

type Detector struct {
	mu                 sync.Mutex
	counters           map[string][]time.Time
	lastBlock          map[string]time.Time
	threshold          int
	blacklistCountries []string
	blacklistHosts     []string
	whitelistHosts     []string
	blockCooldown      time.Duration
}

func NewDetector(cfg config.Config) *Detector {
	threshold := cfg.Thresholds.MaxRequestsPerMinute
	if threshold <= 0 {
		threshold = 100
	}

	return &Detector{
		counters:           make(map[string][]time.Time),
		lastBlock:          make(map[string]time.Time),
		threshold:          threshold,
		blacklistCountries: cfg.Blacklist.Countries,
		blacklistHosts:     cfg.Blacklist.Hostnames,
		whitelistHosts:     cfg.Whitelist.Hostnames,
		blockCooldown:      time.Minute,
	}
}

func (d *Detector) Process(entry *models.AccessLogEntry, country, hostname string) *models.BlockEvent {
	if entry == nil || entry.IP == "" {
		return nil
	}

	now := entry.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isWhitelistedHost(hostname) {
		return nil
	}

	if d.isBlacklistedCountry(country) {
		return d.makeBlockEvent(entry.IP, "BLACKLISTED_COUNTRY", country, hostname, now)
	}

	if d.isBlacklistedHost(hostname) {
		return d.makeBlockEvent(entry.IP, "BLACKLISTED_HOSTNAME", country, hostname, now)
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
		return d.makeBlockEvent(entry.IP, reason, country, hostname, now)
	}

	return nil
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
