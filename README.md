# Cloudflare Access Cookie Proxy

A lightweight HTTP proxy that seamlessly handles Cloudflare Access authentication for client applications that don't natively support it. Perfect for Docker registries, APIs, and other services protected by Cloudflare Access.

## 🎯 Problem Solved

When you protect services with Cloudflare Access, clients like Docker, kubectl, or curl can't authenticate because they don't support the browser-based authentication flow. This proxy bridges that gap by:

- Handling Cloudflare Access authentication transparently
- Acting as a local proxy between your client and the protected service
- Supporting all HTTP methods (GET, POST, PUT, DELETE, etc.)
- Maintaining session persistence with manual cookie management

## 🚀 Features

- **Manual Authentication**: Prompts for CF_Authorization cookies when needed
- **Multi-Protocol Support**: Works with Docker registries, REST APIs, and any HTTP service
- **OCI Compliance**: Special handling for Docker registry manifest requests
- **Session Management**: Tracks cookie expiration and prompts for renewal
- **Easy Setup**: Single Python script with minimal dependencies
- **Docker Ready**: Includes Dockerfile for containerized deployment

## 📋 Requirements

- Python 3.7+
- `requests` library
- Access to a Cloudflare Access protected domain

## 🔧 Installation

### Option 1: Direct Python Usage

```bash
# Clone the repository
git clone https://github.com/yourusername/cloudflare-access-proxy.git
cd cloudflare-access-proxy

# Install dependencies
pip install requests

# Run the proxy
python3 cf_proxy.py your-protected-domain.com --port 8081
```

### Option 2: Docker

```bash
# Build the image
docker build -t cf-proxy .

# Run the proxy
docker run -it --rm -p 8081:8081 cf-proxy your-protected-domain.com --port 8081
```

## 🖥️ Usage

### Basic Usage

1. **Start the proxy:**
   ```bash
   python3 cf_proxy.py registry.yourdomain.com --port 8081
   ```

2. **Authenticate when prompted:**
   
   **Option A: Using cloudflared (recommended):**
   ```bash
   # Login and get token
   cloudflared access login https://registry.yourdomain.com
   export CF_TOKEN=$(cloudflared access token -app=https://registry.yourdomain.com)
   # Paste the token when the proxy prompts
   ```
   
   **Option B: Manual browser method:**
   - Open your browser and navigate to `https://registry.yourdomain.com`
   - Complete the Cloudflare authentication
   - Copy the `CF_Authorization` cookie value from browser dev tools
   - Paste it into the proxy when prompted

3. **Configure your client:**
   - Instead of `registry.yourdomain.com`, use `localhost:8081`

### Docker Registry Example

```bash
# Start the proxy for your Docker registry
python3 cf_proxy.py registry.yourdomain.com

# Configure Docker to use the proxy
# Option 1: Use localhost in commands
docker pull localhost:8081/your-image:tag

# Option 2: Add to /etc/hosts
echo "127.0.0.1 registry.yourdomain.com" >> /etc/hosts
docker pull registry.yourdomain.com/your-image:tag
```

### API Client Example

```bash
# Start the proxy for your API
python3 cf_proxy.py api.yourdomain.com --port 8082

# Use curl through the proxy
curl http://localhost:8082/api/endpoint
```

## 🔐 Authentication Methods

### Recommended: CloudFlared CLI Authentication

The easiest and most secure way to authenticate:

1. **Install cloudflared** (if not already installed):
   ```bash
   # Download from https://github.com/cloudflare/cloudflared/releases
   # Or use package manager:
   brew install cloudflare/cloudflare/cloudflared  # macOS
   ```

2. **Login via cloudflared**:
   ```bash
   cloudflared access login https://your-protected-domain.com
   ```
   This opens your browser, handles authentication, and stores the token securely.

3. **Get the token for the proxy**:
   ```bash
   export CF_TOKEN=$(cloudflared access token -app=https://your-protected-domain.com)
   echo $CF_TOKEN  # Copy this token to paste into the proxy
   ```

4. **Run the proxy** and paste the token when prompted.

**Benefits:**
- ✅ No manual cookie extraction from browser dev tools
- ✅ Same CF_Authorization token as browser cookies
- ✅ Cleaner CLI workflow
- ✅ Automatic token refresh handling

### Manual Authentication (Fallback)

If cloudflared is not available, the proxy can still work with manual cookie extraction:
1. Visit the protected domain in your browser
2. Complete Cloudflare authentication
3. Extract the `CF_Authorization` cookie from browser dev tools
4. Provide it to the proxy when prompted

**Session Management:**
- Tokens/cookies expire after 23 hours (typical CF Access session)
- The proxy will prompt for re-authentication when expired
- Each request checks authentication status automatically

## 🐳 Docker Usage

### Quick Start

```bash
# Interactive mode (recommended for authentication prompts)
docker run -it --rm -p 8081:8081 cf-proxy registry.yourdomain.com

# Background mode (only after successful authentication)
docker run -d -p 8081:8081 --name cf-proxy cf-proxy registry.yourdomain.com
```

### Docker Compose

```yaml
version: '3.8'
services:
  cf-proxy:
    build: .
    ports:
      - "8081:8081"
    command: ["registry.yourdomain.com", "--port", "8081"]
    stdin_open: true
    tty: true
```

## 🔧 Configuration

### Command Line Options

```bash
python3 cf_proxy.py <domain> [--port PORT]
```

- `domain`: Target domain protected by Cloudflare Access
- `--port`: Local proxy port (default: 8081)

### Environment Variables

Currently, all configuration is done via command line arguments. Environment variable support is planned for future releases.

## 🛠️ Use Cases

### Docker Registry

Perfect for private Docker registries behind Cloudflare Access:

```bash
# Start proxy
python3 cf_proxy.py registry.company.com

# Use with Docker commands
docker login localhost:8081
docker push localhost:8081/app:latest
docker pull localhost:8081/app:latest
```

### Kubernetes API

Access Kubernetes clusters protected by Cloudflare Access:

```bash
# Start proxy
python3 cf_proxy.py k8s-api.company.com --port 8443

# Configure kubectl
kubectl config set-cluster company --server=http://localhost:8443
kubectl get pods
```

### REST APIs

Access any REST API behind Cloudflare Access:

```bash
# Start proxy
python3 cf_proxy.py api.company.com

# Make API calls
curl http://localhost:8081/users
curl -X POST http://localhost:8081/data -d '{"key":"value"}'
```

## 🚦 Troubleshooting

### Common Issues

**401 Authentication Error**
- Ensure your CF_Authorization cookie is valid and not expired
- Check that you're accessing the correct domain
- Verify the cookie was copied correctly (no extra spaces)

**Connection Refused**
- Check that the target domain is accessible
- Verify the proxy is running on the correct port
- Ensure no firewall is blocking the connection

**Docker Registry Issues**
- Make sure OCI manifest types are supported (handled automatically)
- Check Docker daemon configuration for localhost registries
- Verify registry endpoint paths are correct

**Cookie Expiration**
- Cookies typically expire after 23 hours
- The proxy will prompt for re-authentication automatically
- Keep the terminal session active to see renewal prompts

### Debug Mode

For verbose logging, modify the script to enable debug output:

```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

## 🤝 Contributing

Contributions are welcome! Areas for improvement:

- Automatic browser cookie extraction
- GUI for easier setup
- Configuration file support
- SSL/TLS termination
- Multiple domain support
- Health check endpoints

## 📄 License

This project is open source and available under the MIT License.

## 🔗 Related Projects

- [Cloudflare Access Documentation](https://developers.cloudflare.com/cloudflare-one/applications/)
- [Docker Registry API](https://docs.docker.com/registry/spec/api/)

## ⚠️ Security Notes

- The proxy handles authentication cookies - ensure it runs in a secure environment
- Cookies are stored in memory only and expire after 23 hours
- Use HTTPS for the target domain to maintain security
- Consider firewall rules to restrict proxy access

## 📊 Status

- ✅ HTTP proxy functionality
- ✅ CloudFlared CLI authentication (recommended)
- ✅ Manual Cloudflare Access cookie handling  
- ✅ Docker registry support with OCI compliance
- ✅ Multi-method support (GET, POST, PUT, DELETE, HEAD)
- ✅ Token/cookie expiration tracking and renewal prompts
- 🔄 Automatic token refresh via cloudflared (planned)
- 🔄 Configuration file support (planned)
- 🔄 Multiple domain support (planned)

---

**Found this useful?** Star the repository and share your experience in the [Cloudflare Community Forum](https://community.cloudflare.com/)!