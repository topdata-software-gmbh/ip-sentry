package parser

import (
	"regexp"
	"time"

	"github.com/topdata/topdata-ip-aggregator/internal/models"
)

var logRegex = regexp.MustCompile(`^(?:(\S+)\s+)?(\d{1,3}(?:\.\d{1,3}){3}|[a-fA-F0-9:]+)\s+\S+\s+\S+\s+\[([^\]]+)\]\s+"(\S+)\s+([^"]+?)\s+[^"]+"\s+\d+\s+\d+\s+"[^"]*"\s+"([^"]*)"`)

func ParseLine(line string) *models.AccessLogEntry {
	matches := logRegex.FindStringSubmatch(line)
	if len(matches) != 7 {
		return nil
	}

	ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[3])
	if err != nil {
		ts = time.Now()
	}

	return &models.AccessLogEntry{
		IP:        matches[2],
		Timestamp: ts,
		Method:    matches[4],
		Path:      matches[5],
		UserAgent: matches[6],
		Host:      matches[1],
	}
}
