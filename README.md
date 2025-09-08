# 📦 LTP Service

A backend service written in Go that provides the **Last Traded Price (LTP)** of Bitcoin against multiple fiat currencies, powered by the public Kraken API.  
The application is fully containerized with Docker and follows a **hexagonal architecture** to ensure scalability, maintainability, and testability.

---

## 🌐 System Overview

- Exposes a REST API on **`/api/v1/ltp`**.
- Retrieves Bitcoin prices in the following pairs:
  - `BTC/USD`
  - `BTC/CHF`
  - `BTC/EUR`
- Supports:
  - Requesting a **single pair** or a **list of pairs**.
  - Accurate data up to the **last minute** using Kraken's "last trade closed" values.
- Includes:
  - **Integration tests** for API and service layers.
  - **Dockerized deployment** for easy setup and execution.

---

## 🛠️ Technical Stack

| Component       | Technology               | Port  |
|-----------------|--------------------------|-------|
| **Backend**     | Go 1.21 + Chi Router     | 8080  |
| **Cache**       | Redis (for LTP caching)  | 6379  |
| **Tests**       | Go + Testify             | —     |
| **Container**   | Docker & Docker Compose  | —     |

---

## 🚀 Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (v20.10+)
- [Docker Compose](https://docs.docker.com/compose/install/) (v2.0+)
- (Optional) Go 1.21+ for local development

---

### Installation & Execution

1. **Clone the repository:**
   ```bash
   git clone https://github.com/FrancoRivero2025/go-exercise.git
   cd go-exercise
   ```

2. **Start the application with Docker Compose:**
   ```bash
   docker-compose up -d --build
   ```

3. **Check logs (optional):**
   ```bash
   docker-compose logs -f ltp-service
   ```

---

### Project Structure

```bash
.
├── cmd/                          # Application entry points
│   └── ltp-service/              # Main service
│       └── main.go               # Initializes dependencies and starts the HTTP server
│
├── config/                       # Centralized configuration
│   ├── config.go                 # Loads and parses configuration from YAML/env
│   └── local.yaml                # Default configuration for local environment
│
├── docker-compose.yml            # Service orchestration (Go API + Redis)
├── Dockerfile                    # Production-ready Docker image
├── Dockerfile.test               # Docker image for running tests with dependencies
│
├── go.mod                        # Go module definition and dependencies
├── go.sum                        # Dependency checksums (lockfile)
│
├── internal/                     # Internal logic (hexagonal architecture)
│   ├── adapters/                 # Infrastructure adapters (inbound/outbound)
│   │   ├── cache/                # Cache implementation
│   │   │   ├── cache.go          # Cache interface
│   │   │   └── redis_cache.go    # Redis-based cache implementation
│   │   │
│   │   ├── http/                 # HTTP adapter (REST API)
│   │   │   ├── handler.go        # API handlers for /api/v1/ltp
│   │   │   └── integration_test.go # Integration tests for the HTTP layer
│   │   │
│   │   ├── kraken/               # External Kraken API client
│   │   │   └── client.go         # Communication with Kraken REST API (ticker endpoint)
│   │   │
│   │   ├── log/                  # Centralized logging
│   │   │   └── logger.go         # Logger configuration and wrapper
│   │   │
│   │   └── refresher/            # Background worker for data refresh
│   │       └── worker.go         # Goroutine that periodically updates cached prices
│   │
│   ├── application/              # Application services (business logic)
│   │   ├── service.go            # Core LTPService implementation (uses ports/domain)
│   │   └── service_test.go       # Unit tests for the service layer
│   │
│   └── domain/                   # Domain entities and interfaces
│       ├── ltp.go                # Domain model (LTP, pairs, etc.)
│       └── mocks/                # Mocks for testing
│           ├── cache.go          # Cache mock
│           └── market_data_provider.go # Market data provider mock
│
├── main.go                       # Alternative root entry point (optional)
├── Makefile                      # Build/test/docker shortcuts (e.g. `make test`)
├── README.md                     # Main project documentation

```

---

### 🔌 API Usage

#### Request all pairs:
```bash
curl http://localhost:8080/api/v1/ltp
```

#### Request specific pair(s):
```bash
curl "http://localhost:8080/api/v1/ltp?pairs=BTC/USD,BTC/EUR"
```

#### Example response:
```json
{
  "ltp": [
    {
      "pair": "BTC/CHF",
      "amount": 49000.12
    },
    {
      "pair": "BTC/EUR",
      "amount": 50000.12
    },
    {
      "pair": "BTC/USD",
      "amount": 52000.12
    }
  ]
}
```

---

## 🧪 Running Tests

Run integration tests locally:

```bash
go test ./... -tags=integration
```

or inside the Docker container:

```bash
# Run integration tests
docker-compose run --rm integration-tests

# Run unit tests
docker-compose run --rm unit-tests

```

---

## 📓 Technical Notes

1. **Architecture**
   - Designed using **Hexagonal Architecture (Ports & Adapters)**.
   - Clear separation between domain, application logic, and infrastructure.
   - Easy to extend with new currency pairs or different data sources.

2. **Caching**
   - Latest prices cached in **Redis**.
   - Ensures low latency and reduces load on Kraken API.
   - Cache TTL ensures values are always accurate within **1 minute**.

3. **Error Handling**
   - Best-effort response: returns successful pairs even if some fail.
   - Proper HTTP status codes and JSON error messages for clients.

4. **Dockerized Services**
   - `ltp-service`: Go API server.
   - `redis`: caching layer.
   - Configurable via environment variables (`local.yaml`).

---


## ⚙️ Maintenance Commands

| Command                               | Description                   |
|---------------------------------------|-------------------------------|
| `docker-compose up -d --build`        | Build & start all services    |
| `docker-compose down -v`              | Stop & clean volumes          |
| `docker-compose logs -f ltp-service`  | Tail API logs                 |
| `go test ./... -tags=integration`     | Run integration tests         |

---

# Golang Developer Assigment

Develop in Go language a service that will provide an API for retrieval of the Last Traded Price of Bitcoin for the following currency pairs:

1. BTC/USD
2. BTC/CHF
3. BTC/EUR


The request path is:
/api/v1/ltp

The response shall constitute JSON of the following structure:
```json
{
  "ltp": [
    {
      "pair": "BTC/CHF",
      "amount": 49000.12
    },
    {
      "pair": "BTC/EUR",
      "amount": 50000.12
    },
    {
      "pair": "BTC/USD",
      "amount": 52000.12
    }
  ]
}

```

# Requirements:

1. The incoming request can done for as for a single pair as well for a list of them
2. You shall provide time accuracy of the data up to the last minute.
3. Code shall be hosted in a remote public repository
4. readme.md includes clear steps to build and run the app
5. Integration tests
6. Dockerized application

# Docs
The public Kraken API might be used to retrieve the above LTP information
[API Documentation](https://docs.kraken.com/rest/#tag/Spot-Market-Data/operation/getTickerInformation)
(The values of the last traded price is called “last trade closed”)

---
