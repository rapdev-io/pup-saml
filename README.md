# pup-saml

RapDev pup extension — SAML and auth discovery for Datadog.

## Install

```bash
pup extension install rapdev-io/pup-saml
```


## Usage

### `pup saml discover`

Full SAML and auth discovery pass. Runs all API calls concurrently and returns a single JSON object covering SAML config, SP metadata, authn_mappings, role summary, user distribution, and service accounts.

```bash
pup saml discover [--org <name>]
```

### `pup saml mappings`

Lists all `authn_mappings` with role names resolved.

```bash
pup saml mappings [--org <name>]
```

## Auth

Credentials are forwarded automatically by pup via environment variables (`DD_ACCESS_TOKEN`, `DD_API_KEY`, `DD_APP_KEY`, `DD_SITE`, `DD_ORG`). No configuration needed beyond a valid `pup auth` session.

## Release

Push a `v*` tag to trigger a GoReleaser build:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

Release assets are standalone binaries: `pup-saml-<os>-<arch>`.
