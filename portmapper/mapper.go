package portmapper

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/akerouanton/docker-cloudflared-mapper/sliceutil"
	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/zero_trust"
	"github.com/docker/go-plugins-helpers/portmapper"
)

var _ portmapper.Driver = (*driver)(nil)

type driver struct {
	client    *cloudflare.Client
	accountId string
}

func newDriver(c *cloudflare.Client, accountId string) *driver {
	return &driver{
		client:    c,
		accountId: accountId,
	}
}

// TODO(aker): should MapPorts be idempotent?
func (d *driver) MapPorts(req portmapper.MapPortsRequest) (_ portmapper.MapPortsResponse, retErr error) {
	ctx := context.Background()

	if err := d.validatePortBindingReqs(req.Reqs); err != nil {
		return portmapper.MapPortsResponse{}, err
	}

	hostname, ok := findLabel(req.Labels, labelHostname)
	if !ok {
		return portmapper.MapPortsResponse{}, fmt.Errorf("label '%s' not found", labelHostname)
	}

	domainName, err := extractDomainName(hostname)
	if err != nil {
		return portmapper.MapPortsResponse{}, err
	}

	tunnelId, ok := findLabel(req.Labels, labelTunnelId)
	if !ok || tunnelId == "" {
		tunnelName, ok := findLabel(req.Labels, labelTunnelName)
		if !ok || tunnelName == "" {
			return portmapper.MapPortsResponse{}, fmt.Errorf("label '%s' is empty or not found, and label '%s' is empty or not found. Specify either one of them", labelTunnelId, labelTunnelName)
		}

		var err error
		tunnelId, err = d.findTunnel(ctx, d.accountId, tunnelName)
		if err != nil {
			return portmapper.MapPortsResponse{}, err
		}
	}

	serviceProto := "http"
	if proto, ok := findLabel(req.Labels, labelProto); ok && proto != "" {
		serviceProto = proto
	}

	path := ""
	if pathLabel, ok := findLabel(req.Labels, labelPath); ok && pathLabel != "" {
		path = fmt.Sprintf("/%s", strings.TrimPrefix(pathLabel, "/"))
	}

	pbReqs := coalesceReqs(req.Reqs)
	ingressConfig := sliceutil.Map(pbReqs, func(pbReq portmapper.PortBindingReq) zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigIngress {
		return zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigIngress{
			Hostname: cloudflare.F(hostname),
			Service:  cloudflare.F(fmt.Sprintf("%s://%s:%d%s", serviceProto, pbReq.BackendIP, pbReq.BackendPort, path)),
		}
	})

	// Add a catch-all ingress rule to return 404 for all other requests. This
	// doesn't make much sense for non-HTTP services, but Cloudflare requires it.
	ingressConfig = append(ingressConfig, zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigIngress{
		Service: cloudflare.F("http_status:404"),
	})

	// Re-configure the tunnel to connect it to the ports.
	tun, err := d.client.ZeroTrust.Tunnels.Cloudflared.Configurations.Update(ctx, tunnelId, zero_trust.TunnelCloudflaredConfigurationUpdateParams{
		AccountID: cloudflare.F(d.accountId),
		Config: cloudflare.F(zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfig{
			Ingress:       cloudflare.F(ingressConfig),
			OriginRequest: cloudflare.F(zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigOriginRequest{}),
		}),
	})
	if err != nil {
		return portmapper.MapPortsResponse{}, fmt.Errorf("failed to update tunnel configuration: %w", err)
	}

	if err := d.insertCNAME(ctx, tun.TunnelID, domainName, hostname); err != nil {
		return portmapper.MapPortsResponse{}, fmt.Errorf("failed to insert CNAME: %w", err)
	}

	// Return the port bindings.
	return portmapper.MapPortsResponse{
		PortBindings: sliceutil.Map(pbReqs, func(req portmapper.PortBindingReq) portmapper.PortBinding {
			return portmapper.PortBinding{
				Proto:       req.Proto,
				BackendIP:   req.BackendIP,
				BackendPort: req.BackendPort,
			}
		}),
	}, nil
}

func (d *driver) validatePortBindingReqs(reqs []portmapper.PortBindingReq) error {
	for _, req := range reqs {
		if !req.FrontendIP.IsUnspecified() {
			return errors.New("frontend IP should not be specified")
		}

		if req.FrontendPort != 0 {
			return errors.New("frontend port should be 0")
		}

		if req.Proto != portmapper.ProtocolTCP {
			return errors.New("only TCP is supported")
		}
	}

	return nil
}

func coalesceReqs(pbReqs []portmapper.PortBindingReq) []portmapper.PortBindingReq {
	// Sort the port binding requests to group together PBs that must be coalesced.
	slices.SortFunc(pbReqs, cmpPBReq)

	var coalesced []portmapper.PortBindingReq
	for i, pbReq := range pbReqs {
		if i > 0 && mustCoalesce(pbReqs[i-1], pbReq) {
			coalesced = slices.Delete(coalesced, i-1, i)
		}

		coalesced = append(coalesced, pbReq)
	}

	return coalesced
}

func cmpPBReq(a, b portmapper.PortBindingReq) int {
	if a.Proto != b.Proto {
		return int(a.Proto) - int(b.Proto)
	}
	c := a.FrontendIP.Unmap().Compare(b.FrontendIP.Unmap())
	return c
}

func mustCoalesce(a, b portmapper.PortBindingReq) bool {
	return a.FrontendIP.Is4() &&
		a.FrontendIP.IsUnspecified() &&
		b.FrontendIP.Is6() &&
		b.FrontendIP.IsUnspecified() &&
		a.FrontendPort == b.FrontendPort &&
		a.FrontendPortEnd == b.FrontendPortEnd
}

func (d *driver) UnmapPorts(req portmapper.UnmapPortsRequest) error {
	ctx := context.Background()

	hostname, ok := findLabel(req.Labels, labelHostname)
	if !ok {
		return fmt.Errorf("label '%s' not found", labelHostname)
	}

	domainName, err := extractDomainName(hostname)
	if err != nil {
		return err
	}

	tunnelId, ok := findLabel(req.Labels, labelTunnelId)
	if !ok || tunnelId == "" {
		tunnelName, ok := findLabel(req.Labels, labelTunnelName)
		if !ok || tunnelName == "" {
			return errors.New("label 'tunnel_id' is empty or not found, and label 'tunnel_name' is empty or not found. Specify either one of them")
		}

		var err error
		tunnelId, err = d.findTunnel(ctx, d.accountId, tunnelName)
		if err != nil {
			return err
		}
	}

	// Remove ingress rules.
	// TODO(aker): remove only ingress rules for the ports that are being unmapped.
	d.client.ZeroTrust.Tunnels.Cloudflared.Configurations.Update(ctx, tunnelId, zero_trust.TunnelCloudflaredConfigurationUpdateParams{
		AccountID: cloudflare.F(d.accountId),
		Config: cloudflare.F(zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfig{
			Ingress: cloudflare.F([]zero_trust.TunnelCloudflaredConfigurationUpdateParamsConfigIngress{
				{
					Service: cloudflare.F("http_status:404"),
				},
			}),
		}),
	})

	// Remove CNAME record
	if err := d.removeCNAMEs(ctx, domainName, hostname); err != nil {
		return fmt.Errorf("failed to remove CNAME: %w", err)
	}

	return nil
}
