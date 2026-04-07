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
