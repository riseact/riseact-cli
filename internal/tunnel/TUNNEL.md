# Dev tunnels

How a partner's `localhost` becomes a public HTTPS URL that loads inside the
Riseact iframe. This replaces ngrok.

## Why it exists

Partners build apps against the Node SDK and run them locally. Riseact renders
those apps in an iframe, and Riseact's servers cannot reach a laptop on a home
or office network. Something has to accept public traffic and forward it to the
developer's machine.

## The pieces

| Component | Where | Role |
| --- | --- | --- |
| `frpc` | inside this CLI | opens the tunnel, forwards traffic to the local reverse proxy |
| `frps` | `riseact-frp` repo, Swarm service | accepts tunnels, routes by hostname |
| authorization plugin | `riseact-core`, `riseact/controlplane/tunnel/` | decides whether a tunnel may exist |
| Caddy | shared `caddy-docker-proxy` | terminates TLS, routes to frps |

Only Caddy is exposed. The frps service publishes no ports at all — it is
reachable solely over the `public` overlay network. That is also what keeps
partners to HTTP tunnels: a TCP or UDP proxy would bind a port inside the frps
container that nothing can route to.

## The subdomain

Each app gets one stable hostname, derived from its own OAuth credentials:

```
subdomain = "d" + base32(hmac_sha256(key=client_secret, msg=client_id))[:26]
```

lowercased, so the result is a valid DNS label. The CLI and riseact-core compute
this independently and must agree byte for byte — a change on one side breaks
tunnels silently, so the two implementations are covered by matching tests.

Because it is derived rather than assigned, there is no registry to keep, and
the URL is stable across restarts. That in turn makes `UpdateAppUris` idempotent
instead of rewriting `redirect_uri` on every `riseact dev`.

## What happens on `riseact dev`

1. The CLI starts its local reverse proxy, merging the app server and the HMR
   websocket into a single origin.
2. It computes the subdomain and connects `frpc` to
   `wss://tunnel.riseact.org/~!frp`. Caddy terminates TLS and forwards to frps,
   whose muxer recognises that path prefix on its control port.
3. Before binding the hostname, frps POSTs to riseact-core. The app's
   `client_id` and `client_secret` travel as frp metadata; core verifies them
   and confirms the requested subdomain is the one that app is entitled to.
4. The CLI requests its own public URL once, to force certificate issuance
   before anyone else arrives, then updates `app_url`.
5. Traffic flows: browser → Caddy → frps → frpc → local reverse proxy → app.

## Authorization

Two gates, deliberately unequal.

The frp token is compiled into the CLI binary that partners download, so treat
it as public. It stops scanners, nothing more, and rotates with each release.

The plugin is the real gate. riseact-core answers per application, which means
credentials can be revoked centrally and a partner can only ever bind their own
subdomain. It is hooked on the `NewProxy` operation only: during a core deploy,
existing tunnels keep working and only new ones are refused. Hooking `Ping` or
`NewUserConn` would put core in the path of every partner request.

It fails closed — frps registers a proxy only when the plugin call succeeds.

## TLS

There is no wildcard certificate and no DNS-provider plugin in Caddy. The tunnel
zone uses **on-demand TLS**: Caddy issues one certificate per hostname at first
request over HTTP-01, the same mechanism already serving `*.riseact.site`.

Caddy consults `on_demand_tls.ask`, which points at
`core.riseact.org/__infra/domain/check/`. That endpoint accepts a tunnel
hostname only while a tunnel is bound to it — `tunnel_authorize` writes the
subdomain into Redis on every successful authorization and the ask endpoint
reads it back. Approving the whole zone instead would let anyone exhaust Let's
Encrypt's weekly quota for `riseact.org`, which is shared with `core`,
`accounts` and `admin`.

**The first request to a new hostname takes about 8 seconds** while the
certificate is issued; the TLS handshake is held open for the duration.
Subsequent requests are ~0.2s and renewals happen in the background. This is why
the CLI warms the certificate itself and says so, instead of leaving a blank
iframe: eight unexplained seconds read as a broken app. It also removes a real
hazard — a request arriving before the tunnel is bound gets a 404 from the ask
endpoint, and a failed issuance puts Caddy into exponential backoff for that
name, up to a day between retries.

## DNS

Nothing to configure. `*.riseact.org` is a wildcard CNAME to the cluster, and
per RFC 4592 a wildcard matches descendants at any depth as long as no closer
node exists in the zone. `tun.riseact.org` is not a node, so everything under it
resolves.

This is exactly why the zone matters: `dev.riseact.org` **is** a node — the docs
site — and its existence stops the wildcard from applying to its children.

## Routing

The riseact stack declares a catch-all `https://` site pointing at core. Caddy
orders site blocks by specificity, so tunnels are matched first:

```
tunnel.riseact.org  →  frps:7000     control channel
core.riseact.org    →  core:8000
*.tun.riseact.org   →  frps:8080     tunnels
(catch-all)         →  core:8000
```

The control hostname sits outside the tunnel zone on purpose, so no partner
subdomain can ever shadow it.

## When it breaks

Work outwards from core, since each layer depends on the previous one.

```bash
# 1. is the app allowed a tunnel at all?
curl -X POST "https://core.riseact.org/__infra/tunnel/authorize/$TOKEN/" \
  -H 'content-type: application/json' \
  -d '{"op":"NewProxy","content":{"subdomain":"<sub>",
       "user":{"metas":{"client_id":"<id>","client_secret":"<secret>"}}}}'
# expected: {"reject": false, "unchange": true}

# 2. may Caddy issue a certificate for the hostname?
curl -o /dev/null -w '%{http_code}\n' \
  "https://core.riseact.org/__infra/domain/check/?domain=<sub>.tun.riseact.org"
# 200 = a tunnel is bound; 404 = it is not, or TUNNEL_DOMAIN is unset
```

Common causes, in rough order of likelihood:

- `TUNNEL_PLUGIN_TOKEN` differs between core and the frp stack → core answers
  404 and every tunnel is refused.
- `TUNNEL_DOMAIN` unset in core → no hostname is ever accepted for a
  certificate, so tunnels bind but have no TLS.
- A `tlsv1 alert internal error` from curl means Caddy could not obtain a
  certificate — almost always the ask endpoint saying no.
- `router config conflict` from frpc means that subdomain is already bound by
  another connection.

A hostname whose tunnel is not running serves a static page telling the partner
to run `riseact dev`.

## Repositories

- `riseact-frp` — the frps service: Dockerfile, `frps.toml`, Swarm stack, and
  `tools/tunnel-subdomain.py` for computing a subdomain by hand.
- `riseact-core` — `riseact/controlplane/tunnel/` (authorization, subdomain
  derivation, on-demand TLS gate) and `controlplane/infra/selectors.py`.
- this repo — `internal/tunnel/`.
