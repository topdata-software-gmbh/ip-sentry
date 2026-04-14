package models

import "time"

type AccessLogEntry struct {
	IP        string
	Timestamp time.Time
	Method    string
	Path      string
	UserAgent string
	Host      string
}

type BlockEvent struct {
	IP      string
	Reason  string
	Country string
	Host    string
}

type DetectionResult struct {
	Event                  *BlockEvent
	Mechanism              string
	WhitelistHostnameMatch bool
	WhitelistIPRangeMatch  bool
	WhitelistIPMatch       bool
}
