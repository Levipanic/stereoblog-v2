# Security contract — StereoDamage v2

Security focus is pragmatic: protect the single-author admin plane, keep public comments abuse-resistant, and safely handle uploads without turning the project into an enterprise security platform.

## Threat model priorities

Most important risks:

1. attacker gains admin publishing/deletion access;
2. CSRF causes owner browser to perform admin writes;
3. brute force against admin secret;
4. upload of dangerous/unexpected files;
5. stored XSS through posts/comments/media metadata;
6. spam/abuse flood in anonymous comments;
7. IP/proxy mistakes make rate limiting ineffective;
8. backup endpoint leaks secrets/data to unauthenticated users;
9. path traversal or arbitrary filesystem access;
10. unsafe database migration/backup causes data loss.

## Admin route visibility

`/admin` should not be linked in public navigation for unauthenticated users.

However, assume an attacker knows `/admin` exists. Security must never depend on route secrecy.

## Admin authentication

Keep the simple single-owner model.

- Secret/password supplied via environment/config secret.
- Never commit it.
- Production must refuse known default secret values.
- Compare credentials in a timing-safe manner where feasible.
- Login endpoint rate limited by trusted client IP.
- On successful login, create cryptographically random high-entropy session token.
- Store only a salted/hash-derived representation in DB.
- Send raw token only in cookie.
- Session expiration enforced server-side.
- Logout invalidates server session.
- Periodically clean expired sessions.

Do not add OAuth/email/magic-link/passkey infrastructure unless separately requested.

## Cookie/session requirements

Production admin cookie:

- `HttpOnly`;
- `Secure`;
- `SameSite=Lax` or stricter if tested UX allows;
- explicit path;
- no sensitive token readable by client JS;
- sane lifetime configurable by env.

Do not store admin secret or session token in localStorage.

## CSRF

All state-changing admin requests require explicit CSRF protection in addition to session cookie.

A session-bound token header similar to v1 is acceptable.

Also validate expected content types and same-origin signals where useful.

Do not rely only on SameSite cookie.

## Admin API authorization

Every endpoint that can:

- create/edit/delete posts;
- upload/delete/manage media;
- approve/reject/delete comments;
- manage mutes;
- generate/download backups;
- read sensitive moderation detail;

must check admin session server-side.

UI hiding is irrelevant to API auth.

## Backup security

Backup download is highly sensitive.

- Admin-authenticated only.
- CSRF protection if generation is stateful; safe authenticated GET/POST semantics should be deliberate.
- Do not include `.env`, raw admin secret, private keys, reverse-proxy config secrets, or unrelated filesystem content.
- Archive paths must be normalized to prevent traversal.
- Set safe content type/disposition.
- Avoid leaving world-readable temporary archives behind.
- Clean temporary artifacts after response/failure.
- Consider request rate limit because backup can be CPU/IO heavy.

## Upload security

Server is authoritative.

Validate:

- max size;
- allowed content/media type policy;
- safe generated filename;
- extension policy;
- no user-supplied path components;
- non-empty file;
- storage stays inside uploads root.

For images, actual image decoding/metadata inspection can strengthen validation where implemented. Do not trust browser `Content-Type` alone.

SVG deserves special care because it can carry active content. v1 allows `.svg`; v2 must either sanitize/serve safely or deliberately restrict SVG after discussing compatibility impact. Existing historical SVG files still need safe serving behavior. Do not silently delete them.

Uploads should be served with sensible headers (`nosniff`, correct type, attachment disposition for generic files where appropriate).

## Post content/XSS

- Canonical rich text is structured, validated data, not arbitrary HTML.
- Supported marks such as links are rendered through controlled Vue components.
- URL schemes must be allowlisted (`http`, `https`, perhaps `mailto` if explicitly desired); reject `javascript:` etc.
- Comments are plain text.
- Media captions/names/alts are text.
- Never use `v-html` on untrusted/canonical content unless the input has gone through a deliberately audited sanitizer and there is no structured alternative.

## Comments/antispam

Preserve v1 defenses before experimenting with simplification:

- challenge token;
- dynamic honeypot;
- request rate limits;
- cooldown/burst checks;
- duplicate/similarity detection;
- suspicious content scoring;
- pending moderation path;
- mute state;
- attempt logging with TTL;
- IP hashing.

Raw IP should not be displayed in admin. If stored at all, document why; default should continue hashed identifiers.

Antispam must not make normal commenting feel like a CAPTCHA-heavy workflow.

## Proxy/IP handling

Production is behind a reverse proxy.

- Trust proxy headers only when configured and only from expected proxy topology.
- Do not blindly trust arbitrary `X-Forwarded-For` from the internet.
- Ensure rate limiting and IP hashing use the correctly derived client IP.
- Include deployment test for this.

## Security headers

At reverse proxy or app layer, keep appropriate headers, including at least:

- `X-Content-Type-Options: nosniff`;
- frame protection (`frame-ancestors` CSP preferred; `X-Frame-Options` fallback);
- sensible `Referrer-Policy`;
- HTTPS/HSTS only when deployment is confirmed HTTPS;
- a practical CSP can be added once Nuxt/editor/media requirements are understood; do not ship a fake CSP that is immediately disabled by broad unsafe directives.

## Secrets in logs/errors

Never log:

- `ADMIN_SECRET`;
- raw session tokens;
- CSRF tokens;
- private environment content;
- full backup archive content/path if it exposes sensitive structure unnecessarily.

Public API errors should not leak stack traces or filesystem paths in production.

## Dependency/security updates

Use maintained stable versions and keep dependency count small.

At release:

- Go vulnerability check if available in toolchain/workflow;
- npm audit/advisory review for runtime dependencies;
- no known critical vulnerabilities ignored without explicit rationale.
