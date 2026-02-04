# Concurrent Load-Balancing Reverse Proxy with Health Monitoring

A production-ready reverse proxy and load balancer written in Go that intelligently distributes HTTP traffic across multiple backend servers while continuously monitoring their health. Supports multiple load-balancing strategies, provides a RESTful Admin API for runtime management, and ensures graceful shutdown.

---

## Features

- **Reverse Proxy** - Built on `net/http/httputil.ReverseProxy` with custom transport configuration
- **Multiple Load Balancing Strategies**
  - **Round-Robin** (`rr`): Evenly distributes requests across healthy backends
  - **Least-Connections** (`lc`): Routes to the backend with fewest active connections
- **Active Health Monitoring**
  - Configurable periodic health checks (default: 15s)
  - Automatic UP/DOWN status tracking
  - Only routes traffic to healthy backends
- **Dynamic Backend Management**
  - Add/remove backends at runtime via REST API
  - No restart required for configuration changes
  - Duplicate detection
- **Production-Ready Resilience**
  - Connection timeouts (dial, response header, read/write)
  - Automatic backend failover
  - Graceful shutdown with request draining
  - Context propagation to cancel backend requests when the client disconnects
- **Admin API**
  - `GET /status` - View all backends (health + connection count)
  - `POST /backends` - Add new backend
  - `DELETE /backends` - Remove backend

---

##  Requirements

- Go 1.25.5 or higher
- No external dependencies (uses only Go standard library)

---

##  Project Structure

```
Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/
├── main.go                  # Application entry point (orchestrates all components)
├── server.go                # Demo backend server (for local testing)
├── config.json              # Configuration file
├── admin/
│   ├── handlers.go          # HTTP handlers for admin API
│   └── models.go            # Request/response models
├── backend/
│   └── backend.go           # Backend server model (health + connection tracking)
├── config/
│   └── config.go            # Configuration loader
├── healthcheck/
│   └── healthcheck.go       # Periodic health checker
├── loadbalancer/
│   └── loadbalancer.go      # LoadBalancer interface
├── proxy/
│   └── proxy.go             # Reverse proxy handler
└── servers/
    └── servers.go           # Server pool (backend management + selection algorithms)
```

---

##  Quick Start

### 1- Start Backend Servers

Open **three separate terminals** and start demo backend servers:

**Terminal 1:**
```bash
go run server.go 8082
```

**Terminal 2:**
```bash
go run server.go 8083
```

**Terminal 3:**
```bash
go run server.go 8084
```

**Expected Output (per backend):**
```
2026/01/24 21:48:40 [Backend-8082] Starting server on http://localhost:8082
```

Each server will log requests it receives:
```
2026/01/24 21:48:53 [Backend-8082] Received request: GET / from [::1]:57881
2026/01/24 21:48:53 [Backend-8082] Response sent successfully
```

### 2- Configure the Load Balancer

Edit `config.json` (or use the default):

```json
{
  "port": 8080,
  "admin_port": 8081,
  "strategy": "lc",
  "health_check_frequency": "15s",
  "backends": [
    "http://localhost:8082",
    "http://localhost:8083",
    "http://localhost:8084"
  ]
}
```

**Configuration Options:**
- `port` - Load balancer listening port (default: 8080)
- `admin_port` - Admin API port (default: 8081)
- `strategy` - Load balancing algorithm: `rr` or `lc`
- `health_check_frequency` - Health check interval (e.g., `5s`, `15s`, `1m`)
- `backends` - List of backend server URLs

### 3- Start the Load Balancer

```bash
go run main.go --config=config.json
```

**Expected Output:**
```
2026/01/24 21:22:29 [INIT] Starting Load Balancer on port 8080 with strategy: lc
2026/01/24 21:22:29 [INIT] Added backend: http://localhost:8082
2026/01/24 21:22:29 [INIT] Added backend: http://localhost:8083
2026/01/24 21:22:29 [INIT] Added backend: http://localhost:8084
2026/01/24 21:22:29 [INIT] Health Checker service starting...
2026/01/24 21:22:29 [HEALTH] Starting initial health check...
2026/01/24 21:22:29 [HEALTH] Checking 3 backends...
2026/01/24 21:22:29 [INIT] Load Balancer listening on :8080
2026/01/24 21:22:29 [INIT] Admin listening on :8081
2026/01/24 21:22:29 [INIT] All services started successfully!
2026/01/24 21:22:29 [STATUS] Backend http://localhost:8082 marked as UP
2026/01/24 21:22:29 [STATUS] Backend http://localhost:8083 marked as UP
2026/01/24 21:22:29 [STATUS] Backend http://localhost:8084 marked as UP
2026/01/24 21:22:44 [HEALTH] Running periodic health check (interval: 15s)
2026/01/24 21:22:44 [HEALTH] Checking 3 backends...
```

### 4- Send Traffic

```bash
curl http://localhost:8080
```

**Expected Response:**
```
Response from Backend-8082
Time: 2026-01-24T21:52:42Z
Path: /
```

---

##  Admin API Reference

**Base URL:** `http://localhost:8081`

### GET `/status` - View All Backends

**Linux/Mac/Git Bash:**
```bash
curl http://localhost:8081/status
```

**Windows PowerShell:**
```powershell
Invoke-RestMethod http://localhost:8081/status | ConvertTo-Json -Depth 3
```

**Expected Response:**
```json
{
  "total_backends": 3,
  "active_backends": 3,
  "backends": [
    {
      "url": "http://localhost:8082",
      "alive": true,
      "current_connections": 0
    },
    {
      "url": "http://localhost:8083",
      "alive": true,
      "current_connections": 0
    },
    {
      "url": "http://localhost:8084",
      "alive": true,
      "current_connections": 0
    }
  ]
}
```

---

### POST `/backends` - Add Backend

**Linux/Mac/Git Bash:**
```bash
curl -X POST http://localhost:8081/backends \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:8085"}'
```

**Windows PowerShell:**
```powershell
$body = @{url = "http://localhost:8085"} | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8081/backends -Method Post -Body $body -ContentType "application/json"
```

**Windows Command Prompt:**
```cmd
curl.exe -X POST http://localhost:8081/backends -H "Content-Type: application/json" -d "{\"url\":\"http://localhost:8085\"}"
```

**Expected Response:**
```
HTTP/1.1 201 Created
Content-Type: application/json
Date: Sat, 24 Jan 2026 21:21:39 GMT
Content-Length: 71

{"message":"Backend added successfully","url":"http://localhost:8085"}
```

**Server Logs:**
```
2026/01/24 21:21:39 [INIT] Added backend: http://localhost:8085
2026/01/24 21:21:39 [ADMIN] Added new backend via Admin API: http://localhost:8085
```

**Notes:**
- Backend is initially marked as `alive=true`
- Health checker will verify status on next interval
- Returns `409 Conflict` if backend already exists

---

### DELETE `/backends` - Remove Backend

**Linux/Mac/Git Bash:**
```bash
curl -X DELETE http://localhost:8081/backends \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:8082"}'
```

**Windows PowerShell:**
```powershell
$body = @{url = "http://localhost:8082"} | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8081/backends -Method Delete -Body $body -ContentType "application/json"
```

**Expected Response:**
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Sat, 24 Jan 2026 21:25:59 GMT
Content-Length: 73

{"message":"Backend removed successfully","url":"http://localhost:8082"}
```

**Server Logs:**
```
2026/01/24 21:25:59 [ADMIN] Removed backend: http://localhost:8082
```

---

##  Load Balancing Strategies

Configure via `strategy` field in `config.json`:

### Round-Robin (`rr`)
- **Algorithm:** Cycles through backends sequentially
- **Use case:** Equal capacity backends, simple distribution
- **Aliases:** `rr`, `RR`, `round-robin`, `Round-Robin`, `Round-robin`

**Example:**
```
Request 1 → Backend-8082
Request 2 → Backend-8083
Request 3 → Backend-8084
Request 4 → Backend-8082 (wraps around)
```

### Least-Connections (`lc`)
- **Algorithm:** Routes to backend with fewest active connections
- **Use case:** Varying request processing times, optimal load distribution
- **Aliases:** `lc`, `LC`, `least-connections`, `Least-Connections`,`Least-connections`
- **Tie-breaking:** Random selection among backends with equal connections

---

##  Health Checking

- **Frequency:** Configured via `health_check_frequency` (default: 15s)
- **Method:** HTTP GET to backend URL
- **Success criteria:** HTTP status code 200-399
- **Failure handling:** Backend marked as DOWN, removed from rotation
- **Recovery:** Automatically marked UP when health check succeeds

**Health Check Logs:**
```
2026/01/24 21:23:44 [HEALTH] Running periodic health check (interval: 15s)
2026/01/24 21:23:44 [HEALTH] Checking 3 backends...
```

**Status Change Logs:**
```
2026/01/24 21:33:45 [STATUS] Backend http://localhost:8089 marked as DOWN 
2026/01/24 21:33:45 [STATUS] Backend http://localhost:8089 marked as UP
```

---

## 🛡️ Resilience & Timeouts

### Server Timeouts (Client → Load Balancer)
```go
ReadTimeout:       10s  // Time to read client request
WriteTimeout:      10s  // Time to write response to client
IdleTimeout:       120s // Keep-alive timeout
ReadHeaderTimeout: 5s   // Time to read request headers
```

### Transport Timeouts (Load Balancer → Backend)
```go
DialTimeout:           5s   // Time to establish backend connection
ResponseHeaderTimeout: 10s  // Time to receive response headers
```

---

##  Graceful Shutdown

Press `Ctrl+C` to initiate graceful shutdown:

**Expected Output:**
```
^C
2026/01/24 18:23:13 [SHUTDOWN] Shutdown signal received, gracefully shutting down...
2026/01/24 18:23:13 [HEALTH] Health checker stopped via context cancellation
2026/01/24 18:23:13 [HEALTH] Health checker shutdown complete
2026/01/24 18:23:13 [SHUTDOWN] Proxy server shutting down...
2026/01/24 18:23:13 [SHUTDOWN] Admin server shutting down...
2026/01/24 18:23:13 [SHUTDOWN] Servers gracefully stopped
```

**Shutdown Process:**
1. Stop accepting new requests
2. Stop health checker (prevents status changes)
3. Drain existing connections (up to 10-second timeout)
4. Shutdown proxy server
5. Shutdown admin server

---

##  Testing Scenarios

### Test Load Balancing
```bash
# Send multiple requests
curl http://localhost:8080
curl http://localhost:8080
curl http://localhost:8080
```

**Backend Logs Show Distribution:**
```
[Backend-8082] Received request: GET / from [::1]:57881
[Backend-8082] Response sent successfully
[Backend-8083] Received request: GET / from [::1]:58003
[Backend-8083] Response sent successfully
[Backend-8084] Received request: GET / from [::1]:58126
[Backend-8084] Response sent successfully
```

**Expected Health Check Logs:**
```
2026/01/24 18:31:12 [HEALTH] Health check failed for http://localhost:8083: Get "http://localhost:8083": dial tcp [::1]:8083: connectex: No connection could be made because the target machine actively refused it.
2026/01/24 18:31:12 [STATUS] Backend http://localhost:8083 marked as DOWN 
```

### Test Dynamic Backend Management
```bash
# Start new backend
go run server.go 8085

# Add to load balancer
curl.exe -X POST http://localhost:8081/backends -H "Content-Type: application/json" -d "{\"url\":\"http://localhost:8085\"}"

#Verify
curl http://localhost:8081/status
```

---

##  Troubleshooting

| Issue | Solution |
|-------|----------|
| **"No healthy backends available"** | Ensure backend servers are running: `go run server.go 8082` |
| **"Invalid request body"** on POST | Use PowerShell syntax: `$body = @{url="..."} \| ConvertTo-Json` |
| **Backend always DOWN** | Check backend responds to GET with 200-399 status |
| **Connection refused** | Verify ports 8080-8084 are not in use by other programs |

---
