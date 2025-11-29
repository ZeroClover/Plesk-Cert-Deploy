## ADDED Requirements
### Requirement: CLI inputs for Plesk certificate deployment
The tool SHALL accept `--subscription/-s` (required), `--name/-n` (required), optional `--module/-m`, and optional `--pub`, `--pri`, `--ca` flags to control certificate deployment targets.

#### Scenario: missing required args
- **WHEN** the user omits either subscription or name
- **THEN** the tool fails with a clear error before running any Plesk command

### Requirement: Module-based path resolution
The tool SHALL support module presets `certd` and `acme.sh`, mapping their environment variables to certificate paths; CLI `--pub`/`--pri` SHALL override module env values, and the tool SHALL error if key or cert paths remain unset.

#### Scenario: module certd env fallback
- **WHEN** the user supplies `--module certd` without explicit file flags
- **THEN** the tool reads `HOST_CRT_PATH`, `HOST_KEY_PATH`, and `HOST_IC_PATH` for cert, key, and CA paths respectively and errors if cert or key are unset

#### Scenario: module acme.sh env fallback
- **WHEN** the user supplies `--module acme.sh` without explicit file flags
- **THEN** the tool reads `CERT_FULLCHAIN_PATH`, `CERT_KEY_PATH`, and `CA_CERT_PATH` for cert, key, and CA paths respectively and errors if cert or key are unset

### Requirement: File readability validation
The tool SHALL verify that provided key and certificate files exist and are readable before proceeding; when a CA path is provided it SHALL also validate readability, failing fast on any invalid path.

#### Scenario: unreadable key file
- **WHEN** the resolved private key path does not exist or cannot be read
- **THEN** the tool exits with an error and does not invoke Plesk commands

### Requirement: Certificate presence detection
The tool SHALL invoke `plesk bin certificate -l` using `-domain <subscription>` or `-admin` when subscription equals `admin`, and SHALL detect whether the requested certificate name already exists in the listing.

#### Scenario: admin subscription listing
- **WHEN** the subscription argument is `admin`
- **THEN** the tool lists certificates with `plesk bin certificate -l -admin` and evaluates existence based on the returned names

### Requirement: Create or update certificate
The tool SHALL create a certificate with `plesk bin certificate -c` when the name is absent, or update it with `plesk bin certificate -u` when present, substituting `-admin` for `-domain <subscription>` when targeting the admin pool, and passing key, cert, and optional CA paths.

#### Scenario: certificate absent
- **WHEN** the certificate name is not found in the listing
- **THEN** the tool invokes `plesk bin certificate -c '<name>' -domain <subscription> -key-file <...> -cert-file <...> [-cacert-file <...>]` (or `-admin`) to create it

#### Scenario: certificate present
- **WHEN** the certificate name is found in the listing
- **THEN** the tool invokes `plesk bin certificate -u '<name>' -domain <subscription> -key-file <...> -cert-file <...> [-cacert-file <...>]` (or `-admin`) to update it

### Requirement: Command output and success reporting
The tool SHALL capture stdout/stderr from Plesk commands, surfacing failures with the command output, and on full success SHALL emit only `Certificate '<name>' deployed successfully to subscription '<subscription>'.`

#### Scenario: command failure
- **WHEN** any Plesk command returns a non-zero status
- **THEN** the tool surfaces the captured output and exits with an error without printing the final success message

#### Scenario: admin pool success message
- **WHEN** the subscription argument is `admin` and all steps succeed
- **THEN** the final success message uses `Certificate '<name>' deployed successfully to admin pool.` to indicate the admin certificate pool target

### Requirement: Static Go binary constraints
The tool SHALL be implemented in Go without CGO, built as a single statically linked executable suitable for distribution.

#### Scenario: static build enforcement
- **WHEN** building the tool
- **THEN** the build uses `CGO_ENABLED=0` and static linking flags to produce one self-contained binary
