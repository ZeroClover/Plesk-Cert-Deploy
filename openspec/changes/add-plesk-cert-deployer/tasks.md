## 1. Implementation
- [x] 1.1 Initialize Go module, enforce CGO disabled, and set up build flags for a static binary.
- [x] 1.2 Implement CLI argument parsing for module, subscription, name, pub/pri/ca paths with clear required/optional validation.
- [x] 1.3 Resolve certificate paths from module-specific environment variables with CLI overrides; error when key/cert are missing.
- [x] 1.4 Validate readability of key and certificate files (and CA when provided) before running Plesk commands.
- [x] 1.5 Implement Plesk certificate listing to detect existing names, branching to create or update with domain/admin flag handling.
- [x] 1.6 Execute create/update commands, capture and display outputs on failure, and emit final success message only when all steps pass.
- [x] 1.7 Add README usage documentation covering modules, required flags, and build/run examples.
