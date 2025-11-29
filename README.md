# Plesk Certificate Deployer

Go CLI for deploying SSL/TLS certificates to Plesk subscriptions or the admin certificate pool. Resolves certificate paths from optional module presets or explicit flags, validates file readability, lists existing certificates, and creates or updates them accordingly.

## Usage

```
plesk-cert-deploy --subscription <name|admin> --name <cert-name> [--module certd|acme.sh] [--pub <cert.pem> --pri <key.pem>] [--ca <ca.pem>]
```

Flags:
- `--subscription, -s` (required): Plesk subscription name or `admin` for the admin pool.
- `--name, -n` (required): Certificate name.
- `--module, -m` (optional): Path presets:
  - `certd`: uses `HOST_CRT_PATH`, `HOST_KEY_PATH`, `HOST_IC_PATH`
  - `acme.sh`: uses `CERT_FULLCHAIN_PATH`, `CERT_KEY_PATH`, `CA_CERT_PATH`
- `--pub`: Public certificate path (overrides module env).
- `--pri`: Private key path (overrides module env).
- `--ca`: CA bundle path (optional; overrides module env).

If both module presets and CLI flags leave the certificate or key path empty, the tool exits with an error. File paths are validated for existence and readability before invoking Plesk.

## Build

Static binary (no CGO):

```
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -extldflags '-static'" -o plesk-cert-deploy .
```

## Notes

- Requires `plesk` CLI available on PATH with permission to manage certificates.
- On success, the tool prints: `Certificate '<name>' deployed successfully to subscription '<subscription>'.`
- On failures, captured Plesk command output is printed to stderr before exiting.
