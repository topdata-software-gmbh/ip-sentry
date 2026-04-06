package models

import "sync/atomic"

type GlobalStats struct {
	LinesProcessed  uint64
	LinesParsed     uint64
	BlocksRequested uint64
	UniqueIPsSeen   uint64
}

func (s *GlobalStats) IncrementProcessed() { atomic.AddUint64(&s.LinesProcessed, 1) }
func (s *GlobalStats) IncrementParsed()    { atomic.AddUint64(&s.LinesParsed, 1) }
func (s *GlobalStats) IncrementBlocks()    { atomic.AddUint64(&s.BlocksRequested, 1) }

func (s *GlobalStats) Processed() uint64 { return atomic.LoadUint64(&s.LinesProcessed) }
func (s *GlobalStats) Parsed() uint64    { return atomic.LoadUint64(&s.LinesParsed) }
func (s *GlobalStats) Blocks() uint64    { return atomic.LoadUint64(&s.BlocksRequested) }
