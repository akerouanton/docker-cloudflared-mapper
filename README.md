# docker-cloudflared-mapper

This an experimental Docker Engine port-mapper that integrates with Cloudflare
Zero Trust Tunnels.

> [!NOTE]  
> This is experimental work based on an unreleased feature of Docker Engine.
> See [[RFC] Custom port-mappers](https://github.com/moby/moby/issues/50259)

## How to use?

#### Cloudflare

Create a Cloudflare account, and register (or transfer) a domain to Clouflare.

Then, create an _Account API Token_:

- Open Cloudflare dashboard, and go to _Manage Account > Account API Tokens_
- Create a _Custom Token_ with the following permissions: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/get-started/create-remote-tunnel-api/#create-an-api-token

Create a tunnel through Cloudflare Dashboard, and run a cloudflared container on your server:

```terminal
$ export CLOUDFLARE_TUNNEL_TOKEN=...
$ docker run -d --restart=always --network host --name cloudflared \
    cloudflare/cloudflared:latest \
    tunnel --no-autoupdate run --token $CLOUDFLARE_TUNNEL_TOKEN
```

#### Docker plugin

First, install and configure the portmapper plugin:

```terminal
$ export CLOUDFLARE_API_TOKEN=...
$ export CLOUDFLARE_ACCOUNT_ID=...
$ docker plugin install --alias cloudflared albinkerouanton006/cloudflared-mapper:latest \
    CLOUDFLARE_API_TOKEN=$CLOUDFLARE_API_TOKEN \
    CLOUDFLARE_ACCOUNT_ID=$CLOUDFLARE_ACCOUNT_ID
```

Then, edit your `/etc/docker/daemon.json` to set the Engine's `default-port-mapper`:

```json
{
    "default-port-mapper": "cloudflared:latest"
}
```

Restart your Engine for that settings to take effect.

#### Start a container

You're now fully set up!

**HTTP Service Example:**

Start a container exposing a webserver:

```terminal
$ docker run --rm -d -p 80/tcp \
    --label "com.cloudflare.portmapper.tunnel_name=docker-cloudflared-mapper - test" \
    --label "com.cloudflare.portmapper.hostname=web.aker.dev" \
    traefik/whoami
```

You can access it from anywhere using `http://web.aker.dev`

**Raw TCP Stream Example:**

Let's try with a raw TCP port:

```terminal
$ docker run --rm -d -p 5432/tcp \
    --label "com.cloudflare.portmapper.tunnel_name=docker-cloudflared-mapper - test" \
    --label "com.cloudflare.portmapper.hostname=db.aker.dev" \
    --label "com.cloudflare.portmapper.proto=tcp" \
    -e POSTGRES_PASSWORD=mysecretpassword \
    -e POSTGRES_USER=admin \
    -e POSTGRES_DB=myapp \
    postgres:15
```

You can connect to that PostgreSQL database from anywhere:

```bash
# Connect via psql using a container (no local psql required)
$ docker run -it --rm postgres:15 psql -h db.aker.dev -p 5432 -U admin -d myapp
# Password: mysecretpassword
```

## TODO

- [ ] Improve the output of `docker ps`
