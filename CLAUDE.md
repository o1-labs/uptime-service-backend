# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Environment Setup

This project requires Nix for consistent builds and development. All commands should be run inside the nix-shell:

```bash
$ nix-shell
[nix-shell]$ # run commands here
```

## Common Commands

### Building
- `make` - Build the delegation backend binary
- `make clean` - Remove build artifacts
- `make docker` - Build Docker image (requires TAG environment variable)
- `make tidy` - Run go mod tidy

### Testing
- `make test` - Run unit tests for delegation_backend
- `make integration-test` - Run integration tests (requires UPTIME_SERVICE_SECRET)

### Database Operations
- `make db-migrate-up` - Run database migrations up
- `make db-migrate-down` - Run database migrations down

## Architecture Overview

### Core Components

**Delegation Backend Service** (`src/delegation_backend/`)
- HTTP service listening on port 8080 that accepts block producer uptime submissions
- Main entry point: `src/cmd/delegation_backend/main_bpu.go`
- Configuration: `app_config.go` handles environment variables and JSON config files

**ITN Uptime Analyzer** (`src/itn_uptime_analyzer/`)
- Separate component for analyzing uptime data

**Database Migration Tool** (`src/cmd/db_migration/`)
- Handles Cassandra/AWS Keyspaces table creation and migrations

### Key Modules

**Storage Backends** - The service supports multiple storage backends simultaneously:
- AWS S3 (`aws.go`)
- AWS Keyspaces/Cassandra (`aws_keyspaces.go`) 
- PostgreSQL (`postgres.go`)
- Local filesystem

**Authentication & Authorization**:
- Whitelist management via Google Sheets integration (`sheets.go`)
- Signature verification using Mina's reference signer (`signer.go`)
- Rate limiting per public key (`time_heap.go`)

**Data Processing**:
- Block and submission validation (`data.go`, `submit.go`)
- JSON payload unmarshaling and validation (`unmarshal_test.go`)

### Configuration

The service can be configured via:
1. JSON configuration file (set `CONFIG_FILE` environment variable)
2. Environment variables (see README.md for complete list)

Key configuration areas:
- Network settings (`CONFIG_NETWORK_NAME`)
- Storage backend selection (AWS, PostgreSQL, filesystem)  
- Whitelist management (Google Sheets integration)
- Rate limiting (`REQUESTS_PER_PK_HOURLY`)

### API Endpoints

- `POST /v1/submit` - Primary endpoint for block producer submissions
- `/health` - Health check endpoint
- `/` - Basic service identifier

### Build System

The build process uses `scripts/build.sh` which:
- Builds the C reference signer library
- Compiles Go binaries for different components
- Supports Docker image creation
- Handles test execution

### Testing Strategy

- Unit tests: Focus on individual component logic
- Integration tests: Full end-to-end testing with Docker containers
- Test data: Located in `test/data/` with various payload samples

## CI/CD Pipeline

The repository uses GitHub Actions for continuous integration and deployment:

### Workflows
- **Build** (`.github/workflows/build.yml`) - Runs on PR and main branch pushes
  - Builds binary and Docker image
  - Runs unit tests
  - Pushes to GitHub Container Registry (`ghcr.io`) on main branch
- **Integration** (`.github/workflows/integration.yml`) - Full integration testing with Minimina
- **Publish** (`.github/workflows/publish.yaml`) - Triggered by tags or manual dispatch
  - Builds and publishes tagged releases to GitHub Container Registry

### Container Registry
- **Registry**: GitHub Container Registry (`ghcr.io`)  
- **Image**: `ghcr.io/minafoundation/uptime-service-backend`
- **Tags**: `latest` (main branch), commit SHA, and version tags

**Note**: Recent changes migrated from AWS ECR to GitHub Container Registry and updated all workflows to use standard `ubuntu-latest` runners instead of custom MinaFoundation runners.

## Development Notes

- All Go code is in the `src/` directory
- The project uses Go modules (`src/go.mod`)
- External C dependencies are managed via the c-reference-signer submodule
- Database schema migrations are in `database/migrations/`
- Docker configuration in `dockerfiles/`