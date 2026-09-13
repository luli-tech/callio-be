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

## Local Jasmin SMS Gateway

Start Jasmin and its dependencies:

```bash
make jasmin-up
```

Create the local HTTP API user configured in `.env`:

```bash
make jasmin-user
```

The local development credentials are:

```env
JASMIN_BASE_URL=http://localhost:1401
JASMIN_USERNAME=callio_user
JASMIN_PASSWORD=callio_secret
```

Jasmin still needs an SMPP connector and route from a real SMS provider before messages can leave your machine.

## Available Make Commands

- `make build` - Builds the binary into `bin/api`
- `make run` - Starts the application
- `make test` - Runs test suite with race detector enabled
- `make tidy` - Cleans up and verifies `go.mod` dependencies
- `make jasmin-up` - Starts Redis, RabbitMQ, and Jasmin
- `make jasmin-user` - Creates the local Jasmin HTTP API user
- `make clean` - Removes generated build binaries
