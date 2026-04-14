package models

import (
	"sort"
	"sync"
	"sync/atomic"
)

type GlobalStats struct {
	LinesProcessed  uint64
	LinesParsed     uint64
	BlocksRequested uint64
	UniqueIPsSeen   uint64

	WhitelistHostnameHitsCount uint64
	WhitelistIPRangeHitsCount  uint64
	BlocksByCountry            uint64
	BlocksByHostname           uint64
	BlocksByUserAgent          uint64
	BlocksByRateLimit          uint64
	BlocksByOther              uint64

	mu         sync.Mutex
	countries  map[string]uint64
	userAgents map[string]uint64
}

type TopItem struct {
	Key   string
	Count uint64
}

func (s *GlobalStats) IncrementProcessed() { atomic.AddUint64(&s.LinesProcessed, 1) }
func (s *GlobalStats) IncrementParsed()    { atomic.AddUint64(&s.LinesParsed, 1) }
func (s *GlobalStats) IncrementBlocks()    { atomic.AddUint64(&s.BlocksRequested, 1) }
func (s *GlobalStats) IncrementWhitelistHostnameHits() {
	atomic.AddUint64(&s.WhitelistHostnameHitsCount, 1)
}
func (s *GlobalStats) IncrementWhitelistIPRangeHits() {
	atomic.AddUint64(&s.WhitelistIPRangeHitsCount, 1)
}

func (s *GlobalStats) IncrementBlocksByMechanism(mechanism string) {
	switch mechanism {
	case "BLACKLISTED_COUNTRY":
		atomic.AddUint64(&s.BlocksByCountry, 1)
	case "BLACKLISTED_HOSTNAME":
		atomic.AddUint64(&s.BlocksByHostname, 1)
	case "BLACKLISTED_USER_AGENT":
		atomic.AddUint64(&s.BlocksByUserAgent, 1)
	case "RATE_LIMIT_EXCEEDED":
		atomic.AddUint64(&s.BlocksByRateLimit, 1)
	default:
		atomic.AddUint64(&s.BlocksByOther, 1)
	}
}

func (s *GlobalStats) RecordRequest(country, userAgent string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.countries == nil {
		s.countries = make(map[string]uint64)
		s.userAgents = make(map[string]uint64)
	}
	if country != "" && country != "-" {
		s.countries[country]++
	}
	// Many access logs output "-" when UA is empty
	if userAgent != "" && userAgent != "-" {
		s.userAgents[userAgent]++
	}
}

func (s *GlobalStats) TopCountries(n int) []TopItem {
	return s.getTop(s.countries, n)
}

func (s *GlobalStats) TopUserAgents(n int) []TopItem {
	return s.getTop(s.userAgents, n)
}

func (s *GlobalStats) getTop(m map[string]uint64, n int) []TopItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]TopItem, 0, len(m))
	for k, v := range m {
		items = append(items, TopItem{Key: k, Count: v})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})
	if len(items) > n {
		items = items[:n]
	}
	return items
}

func (s *GlobalStats) Processed() uint64 { return atomic.LoadUint64(&s.LinesProcessed) }
func (s *GlobalStats) Parsed() uint64    { return atomic.LoadUint64(&s.LinesParsed) }
func (s *GlobalStats) Blocks() uint64    { return atomic.LoadUint64(&s.BlocksRequested) }
func (s *GlobalStats) WhitelistHostnameHits() uint64 {
	return atomic.LoadUint64(&s.WhitelistHostnameHitsCount)
}
func (s *GlobalStats) WhitelistIPRangeHits() uint64 {
	return atomic.LoadUint64(&s.WhitelistIPRangeHitsCount)
}
func (s *GlobalStats) BlockedByCountry() uint64  { return atomic.LoadUint64(&s.BlocksByCountry) }
func (s *GlobalStats) BlockedByHostname() uint64 { return atomic.LoadUint64(&s.BlocksByHostname) }
func (s *GlobalStats) BlockedByUserAgent() uint64 {
	return atomic.LoadUint64(&s.BlocksByUserAgent)
}
func (s *GlobalStats) BlockedByRateLimit() uint64 {
	return atomic.LoadUint64(&s.BlocksByRateLimit)
}
func (s *GlobalStats) BlockedByOther() uint64 { return atomic.LoadUint64(&s.BlocksByOther) }
