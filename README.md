# Cloudflare Access Proxy

A lightweight HTTP proxy that sits between client applications and Cloudflare Access protected services, handling cookie authentication transparently.

## What it does

This proxy automatically handles Cloudflare Access authentication by:
- Authenticating via `cloudflared` CLI or manual cookie input
- Injecting CF_Authorization cookies into all proxied requests
- Supporting OCI manifest types for Docker registry endpoints
- Preserving request/response headers and body content

## Usage

### Go Version (Recommended)
```bash
# Build the proxy
go build -o cf-proxy cf_proxy.go

# Run with default port (8081)
./cf-proxy registry.yourdomain.com

# Run with custom port
./cf-proxy registry.yourdomain.com --port 8080
```

### Python Version
```bash
# Run with default port (8081)
python3 cf_proxy_original.py registry.yourdomain.com

# Run with custom port
python3 cf_proxy_original.py registry.yourdomain.com --port 8080
```

## Authentication Methods

1. **Cloudflared CLI** (Recommended)
   - Automatically runs `cloudflared access login` and retrieves token
   
2. **Manual Token Input**
   - Run `cloudflared access login https://yourdomain.com`
   - Run `cloudflared access token -app=https://yourdomain.com`
   - Paste the token when prompted

3. **Browser Cookie**
   - Navigate to your protected domain in browser
   - Complete Cloudflare authentication
   - Extract `CF_Authorization` cookie from dev tools
   - Paste when prompted

## Example Use Case

Configure Docker to use the proxy for a Cloudflare-protected registry:
```bash
# Start proxy
./cf-proxy registry.company.com --port 8081

# Configure Docker daemon or client
# Use localhost:8081 instead of registry.company.com
docker pull localhost:8081/myapp:latest
```

## Requirements

- Go 1.21+ (for Go version)
- Python 3.6+ (for Python version)  
- `cloudflared` CLI tool (optional, for automatic auth)