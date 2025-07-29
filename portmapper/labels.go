package portmapper

const (
	labelPrefix     = "com.cloudflare.portmapper."
	labelTunnelId   = labelPrefix + "tunnel_id"
	labelTunnelName = labelPrefix + "tunnel_name"
	labelHostname   = labelPrefix + "hostname"
	labelProto      = labelPrefix + "proto"
	labelPath       = labelPrefix + "path"
)

func findLabel(labels map[string]string, key string) (string, bool) {
	for k, v := range labels {
		if k == key {
			return v, true
		}
	}
	return "", false
}
