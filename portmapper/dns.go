package portmapper

import (
	"context"
	"fmt"
	"strings"

	"github.com/akerouanton/docker-cloudflared-mapper/sliceutil"
	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

func extractDomainName(hostname string) (string, error) {
	unsuf := strings.TrimSuffix(hostname, ".")
	parts := strings.Split(unsuf, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid hostname: %s", hostname)
	}
	return strings.Join(parts[len(parts)-2:], "."), nil
}

func (d *driver) insertCNAME(ctx context.Context, tunnelId, domainName, hostname string) error {
	zoneId, err := d.findZoneId(ctx, domainName)
	if err != nil {
		return fmt.Errorf("failed to find zone %s: %w", domainName, err)
	}

	if hostname == domainName {
		hostname = "@" // Zone apex
	}

	if _, err = d.client.DNS.Records.New(ctx, dns.RecordNewParams{
		ZoneID: cloudflare.F(zoneId),
		Body: dns.RecordNewParamsBody{
			Name:    cloudflare.F(hostname),
			TTL:     cloudflare.F(dns.TTL1), // Automatic TTL
			Type:    cloudflare.F(dns.RecordNewParamsBodyTypeCNAME),
			Content: cloudflare.F(tunnelId + ".cfargotunnel.com"),
			Proxied: cloudflare.F(true),
		},
	}); err != nil {
		return fmt.Errorf("failed to insert CNAME: %w", err)
	}

	return nil
}

func (d *driver) findZoneId(ctx context.Context, domainName string) (string, error) {
	zones, err := d.client.Zones.List(ctx, zones.ZoneListParams{
		Account: cloudflare.F(zones.ZoneListParamsAccount{
			ID: cloudflare.F(d.accountId),
		}),
		Name: cloudflare.F(domainName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to find zone %s: %w", domainName, err)
	}

	if len(zones.Result) == 0 {
		return "", fmt.Errorf("no zone found for %s", domainName)
	}

	return zones.Result[0].ID, nil
}

func (d *driver) removeCNAMEs(ctx context.Context, domainName, hostname string) error {
	zoneId, err := d.findZoneId(ctx, domainName)
	if err != nil {
		return fmt.Errorf("failed to find zone %s: %w", domainName, err)
	}

	if hostname == domainName {
		hostname = "@" // Zone apex
	}

	recordIds, err := d.findCNAMEs(ctx, domainName, hostname)
	if err != nil {
		return fmt.Errorf("failed to find CNAMEs: %w", err)
	}

	for _, recordId := range recordIds {
		if _, err = d.client.DNS.Records.Delete(ctx, recordId, dns.RecordDeleteParams{
			ZoneID: cloudflare.F(zoneId),
		}); err != nil {
			// slog.Error("failed to delete CNAME", "recordId", recordId, "hostname", hostname, "error", err)
			return fmt.Errorf("failed to delete CNAME %s: %w", recordId, err)
		}
	}

	return nil
}

func (d *driver) findCNAMEs(ctx context.Context, domainName, hostname string) ([]string, error) {
	zoneId, err := d.findZoneId(ctx, domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to find zone %s: %w", domainName, err)
	}

	records, err := d.client.DNS.Records.List(ctx, dns.RecordListParams{
		ZoneID: cloudflare.F(zoneId),
		Name: cloudflare.F(dns.RecordListParamsName{
			Exact: cloudflare.F(hostname),
		}),
		Type: cloudflare.F(dns.RecordListParamsTypeCNAME),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find CNAME: %w", err)
	}

	return sliceutil.Map(records.Result, func(r dns.RecordResponse) string {
		return r.ID
	}), nil
}
