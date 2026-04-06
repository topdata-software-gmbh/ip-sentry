package parser

import (
	"regexp"
	"time"

	"github.com/topdata/topdata-ip-aggregator/internal/models"
)

var logRegex = regexp.MustCompile(`^(\S+)\s+\S+\s+\S+\s+\[([^\]]+)\]\s+"(\S+)\s+([^"]+?)\s+[^"]+"\s+\d+\s+\d+\s+"[^"]*"\s+"([^"]*)"`)

func ParseLine(line string) *models.AccessLogEntry {
	matches := logRegex.FindStringSubmatch(line)
	if len(matches) != 6 {
		return nil
	}

	ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
	if err != nil {
		ts = time.Now()
	}

	return &models.AccessLogEntry{
		IP:        matches[1],
		Timestamp: ts,
		Method:    matches[3],
		Path:      matches[4],
		UserAgent: matches[5],
	}
}
