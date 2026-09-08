# Deployment and operations — tiny VPS first

## Goal

Run v2 cheaply and predictably on the same class of tiny personal VPS as v1. Do not require managed cloud services.

## Recommended process model

Replace PM2-as-the-only-process-manager with native Linux services.

Recommended:

```text
systemd
  - stereodamage-api.service  -> compiled Go binary
  - stereodamage-web.service  -> Nuxt/Nitro node server
```

Why:

- Go does not need PM2;
- systemd is already part of most VPS distributions;
- restarts on failure/boot;
- logs available through journald;
- no container requirement;
- low overhead.

PM2 may still technically run the Nuxt Node process, but using one process manager for both services is operationally simpler. Prefer systemd unless existing host constraints strongly favor otherwise.

## Docker

Docker is optional, not a launch requirement.

Do not make Docker the only supported deployment path unless the owner later chooses it.

Reasons to defer:

- the system is only two local processes + persistent files;
- owner values simplicity and low resource use;
- direct systemd deployment is straightforward.

A Docker Compose setup can be added later as an alternative once the native deployment is stable.

## Reverse proxy

Use the owner's existing HTTPS reverse proxy if one already exists.

Conceptual routing:

```text
/api/*     -> 127.0.0.1:<go-port>
/uploads/* -> chosen static handler (Go or proxy)
/*         -> 127.0.0.1:<nuxt-port>
```

Do not expose Go/Nuxt ports publicly if the reverse proxy can bind them to localhost.

## Nuxt -> Go server-side access

Nuxt SSR should use an internal backend base URL such as `http://127.0.0.1:<go-port>` from runtime config, while browser-side requests use same-origin `/api/v1/...`.

Do not hardcode public domain names in application source.

## Suggested runtime paths

Keep persistent state separate from build output conceptually:

```text
/opt/stereodamage/current/   # deployed code/builds
/var/lib/stereodamage/data/blog.db
/var/lib/stereodamage/uploads/
```

or retain project-local `data/` + `uploads/` if that is materially simpler for the owner's VPS.

Whichever path is chosen:

- configure it explicitly;
- ensure service user permissions are correct;
- do not let deploy replace/delete persistent directories;
- backup knows exact paths.

## Build/deploy approach

Keep manual deployment viable.

Example conceptual release:

1. build/test on local/CI environment;
2. build Go binary for VPS architecture;
3. build Nuxt production output;
4. transfer release;
5. stop/replace services during planned cutover;
6. point/copy production DB/uploads into configured persistent paths;
7. start Go so migrations run safely;
8. start Nuxt;
9. health/smoke checks;
10. reload reverse proxy if config changed.

Do not require a CI/CD system for launch, though scripts may make repeat deploys easier.

## First v2 production cutover

This is a one-shot replacement after v2 is proven on backups, not a live dual-backend migration.

Before cutover:

- make a fresh external production backup;
- verify restore procedure;
- record current v1 process/reverse-proxy config;
- know exact rollback command;
- rehearse with a copy.

Cutover should have a short maintenance/write-free window if that simplifies final DB/media synchronization.

## Rollback

Keep v1 deployment/release available.

If v2 fails immediately:

1. stop v2 services;
2. restore untouched pre-cutover DB/media backup if v2 performed writes/migrations that make you uncertain;
3. start v1;
4. restore reverse-proxy routing;
5. investigate offline.

Because migrations are designed additive and v1 tolerates extra columns in the known schema, rollback may be simpler, but never assume this without rehearsal.

## Backup operations

Two levels:

### Admin portable backup

Owner clicks Download Backup and gets DB + uploads + manifest.

### Server/operator backup

Continue supporting simple filesystem/server backup workflows for disaster recovery.

Backups must live off-VPS eventually; a backup stored only on the same cheap VPS does not protect against VPS loss.

## Health checks

Provide lightweight endpoints/process checks:

- Go API health (process + DB reachable, not leaking sensitive info);
- Nuxt HTTP route reachable;
- optional deploy smoke script checks homepage, a post, API health, uploads.

## Logs

Use journald initially.

Useful commands should be documented after service names are final, e.g.:

```text
systemctl status stereodamage-api
journalctl -u stereodamage-api -f
systemctl status stereodamage-web
journalctl -u stereodamage-web -f
```

Do not require an external log vendor.

## Resource awareness

Tiny VPS means:

- avoid multiple redundant Node worker processes;
- use one Nuxt node server unless measurement proves more needed;
- keep Go SQLite pool small/sensible;
- avoid in-memory caches with unbounded growth;
- stream backup/media operations rather than loading huge archives into memory;
- avoid server-side image processing that causes pathological memory spikes; bound concurrency.
