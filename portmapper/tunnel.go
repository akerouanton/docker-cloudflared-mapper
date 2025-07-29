package portmapper

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/zero_trust"
)

func (d *driver) findTunnel(ctx context.Context, accountId string, tunnelName string) (string, error) {
	tunnels, err := d.client.ZeroTrust.Tunnels.List(ctx, zero_trust.TunnelListParams{
		AccountID: cloudflare.F(accountId),
		Name:      cloudflare.F(tunnelName),
		IsDeleted: cloudflare.F(false),
	})
	if err != nil {
		return "", err
	}

	if len(tunnels.Result) == 0 {
		return "", fmt.Errorf("tunnel %q not found", tunnelName)
	}

	return tunnels.Result[0].ID, nil
}
