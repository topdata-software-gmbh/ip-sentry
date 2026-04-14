package whitelist

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	CloudflareIPv4URL = "https://www.cloudflare.com/ips-v4"
	CloudflareIPv6URL = "https://www.cloudflare.com/ips-v6"
	fetchTimeout      = 15 * time.Second
)

// FetchCloudflareRanges fetches Cloudflare IP ranges (both v4 and v6) and
// returns them as parsed CIDR networks.
func FetchCloudflareRanges(ctx context.Context, logger *zap.Logger) ([]*net.IPNet, error) {
	var allNets []*net.IPNet

	for _, url := range []string{CloudflareIPv4URL, CloudflareIPv6URL} {
		nets, err := fetchCIDRList(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", url, err)
		}
		logger.Info("Fetched Cloudflare IP ranges", zap.String("url", url), zap.Int("count", len(nets)))
		allNets = append(allNets, nets...)
	}

	return allNets, nil
}

func fetchCIDRList(ctx context.Context, url string) ([]*net.IPNet, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return parseCIDRList(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
}

func parseCIDRList(r io.Reader) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		_, ipNet, err := net.ParseCIDR(line)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", line, err)
		}
		nets = append(nets, ipNet)
	}
	return nets, scanner.Err()
}

// ParseStaticRanges parses a list of CIDR strings into net.IPNet values.
func ParseStaticRanges(cidrs []string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		nets = append(nets, ipNet)
	}
	return nets, nil
}
