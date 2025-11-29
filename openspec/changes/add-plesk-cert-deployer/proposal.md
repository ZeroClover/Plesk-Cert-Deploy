# Change: Add CLI for deploying SSL/TLS certificates to Plesk

## Why
- Operators need a repeatable way to deploy certificates to Plesk subscriptions or the admin pool using existing issuance pipelines.
- Current repo lacks any tooling to discover existing certs and update or create them automatically.

## What Changes
- Introduce a Go-based CLI that accepts subscription, certificate name, optional module presets, and optional explicit key/cert/CA paths.
- Resolve certificate paths from module-specific environment variables with CLI overrides, validating readability before proceeding.
- Detect whether a certificate name already exists in Plesk and branch to create or update with the correct domain/admin flags.
- Capture shell output and emit a single success message on completion; surface errors when commands fail.

## Impact
- Affected specs: plesk-cert-deploy
- Affected code: new Go CLI entrypoint and supporting modules for input resolution, Plesk command execution, and logging.
