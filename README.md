# twilio-Boss (callio-be)

Backend service built with Go.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.25 or higher
- [MongoDB](https://www.mongodb.com/docs/manual/installation/) 7 or higher
- [Redis](https://redis.io/docs/latest/operate/oss_and_stack/install/archive/install-redis/) 7 or higher

## Getting Started

1. **Clone the repository** (if not already done):
   ```bash
   git clone git@github.com:luli-tech/twilio-Boss.git
   cd twilio-Boss
   ```

2. **Configure environment variables**:
   ```bash
   cp .env.example .env
   ```

3. **Run the application**:
   ```bash
   docker compose up -d mongodb redis
   ```

   ```bash
   make run
   # or
   go run ./cmd/api
   ```

4. **Verify health endpoint**:
   ```bash
   curl http://localhost:8080/health
   ```

## Available Make Commands

- `make build` - Builds the binary into `bin/api`
- `make run` - Starts the application
- `make test` - Runs test suite with race detector enabled
- `make tidy` - Cleans up and verifies `go.mod` dependencies
- `make clean` - Removes generated build binaries
