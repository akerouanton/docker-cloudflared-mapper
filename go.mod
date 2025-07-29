module github.com/akerouanton/docker-cloudflared-mapper

go 1.24.1

require (
	github.com/cloudflare/cloudflare-go/v4 v4.6.0
	github.com/docker/go-plugins-helpers v0.0.0-20240701071450-45e2431495c8
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/sys v0.10.0 // indirect
)

replace github.com/docker/go-plugins-helpers => github.com/akerouanton/go-plugins-helpers v0.0.0-20250723155231-63394c2ea48f
