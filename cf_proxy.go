package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"strconv"

	"github.com/sirupsen/logrus"
)

// CloudflareAccessProxy represents the proxy instance
type CloudflareAccessProxy struct {
	targetDomain  string
	localPort     int
	session       *http.Client
	cfCookie      string
	cookieExpires int64
	logger        *logrus.Logger
}

// NewCloudflareAccessProxy creates a new proxy instance
func NewCloudflareAccessProxy(targetDomain string, localPort int) *CloudflareAccessProxy {
	return &CloudflareAccessProxy{
		targetDomain: targetDomain,
		localPort:    localPort,
		session: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
		logger: logrus.New(),
	}
}

// authenticateWithCloudflared authenticates using cloudflared CLI if available
func (p *CloudflareAccessProxy) authenticateWithCloudflared() bool {
	p.logger.Info("🔐 Authenticating with cloudflared...")
	p.logger.Infof("📡 Target domain: %s", p.targetDomain)

	// Check if cloudflared is available
	if _, err := exec.LookPath("cloudflared"); err != nil {
		p.logger.Error("❌ cloudflared not found in PATH")
		return false
	}

	// Login with cloudflared
	p.logger.Info("🚀 Launching cloudflared access login...")
	loginCmd := exec.Command("cloudflared", "access", "login", fmt.Sprintf("https://%s", p.targetDomain))
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr

	if err := loginCmd.Run(); err != nil {
		p.logger.Errorf("❌ cloudflared login failed: %v", err)
		return false
	}

	// Get the token
	p.logger.Info("🔑 Retrieving access token...")
	tokenCmd := exec.Command("cloudflared", "access", "token", fmt.Sprintf("-app=https://%s", p.targetDomain))
	tokenCmd.Stderr = os.Stderr

	output, err := tokenCmd.Output()
	if err != nil {
		p.logger.Errorf("❌ Failed to retrieve token: %v", err)
		return false
	}

	token := strings.TrimSpace(string(output))
	if token != "" {
		p.cfCookie = token
		// Set expiration to 23 hours from now (typical CF Access session)
		p.cookieExpires = time.Now().Unix() + int64(23*3600)
		p.logger.Info("✅ Successfully authenticated via cloudflared!")
		return true
	}

	return false
}

// extractCFCookieFromBrowser attempts to extract CF_Authorization cookie from browser
func (p *CloudflareAccessProxy) extractCFCookieFromBrowser() bool {
	// TODO: Implement browser cookie extraction
	// Could use a library like github.com/tebeka/selenium or similar
	p.logger.Info("⚠️  Browser cookie extraction not implemented")
	return false
}

// authenticateManual handles manual authentication
func (p *CloudflareAccessProxy) authenticateManual() bool {
	fmt.Printf("\n🔐 Manual authentication for %s\n", p.targetDomain)
	fmt.Println("Choose your method:")
	fmt.Println("1. CloudFlared CLI (recommended)")
	fmt.Println("2. Browser cookie extraction")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nChoose method (1 or 2): ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)

	if choice == "1" {
		fmt.Println("\n📋 CloudFlared steps:")
		fmt.Printf("1. Run: cloudflared access login https://%s\n", p.targetDomain)
		fmt.Printf("2. Run: cloudflared access token -app=https://%s\n", p.targetDomain)
		fmt.Println("3. Copy the token output and paste it below:")

		fmt.Print("\nCF_Authorization token: ")
		tokenInput, _ := reader.ReadString('\n')
		tokenValue := strings.TrimSpace(tokenInput)

		if tokenValue != "" {
			p.cfCookie = tokenValue
			// Set expiration to 23 hours from now (typical CF Access session)
			p.cookieExpires = time.Now().Unix() + int64(23*3600)
			return true
		}
	} else {
		fmt.Println("\n📋 Browser cookie steps:")
		fmt.Printf("1. Open your browser and go to: https://%s\n", p.targetDomain)
		fmt.Println("2. Complete Cloudflare authentication")
		fmt.Println("3. Open browser dev tools → Application → Cookies")
		fmt.Printf("4. Find 'CF_Authorization' cookie for %s\n", p.targetDomain)
		fmt.Println("5. Copy the cookie value and paste it here:")

		fmt.Print("\nCF_Authorization cookie: ")
		cookieInput, _ := reader.ReadString('\n')
		cookieValue := strings.TrimSpace(cookieInput)

		if cookieValue != "" {
			p.cfCookie = cookieValue
			// Set expiration to 23 hours from now (typical CF Access session)
			p.cookieExpires = time.Now().Unix() + int64(23*3600)
			return true
		}
	}

	return false
}

// isAuthenticated checks if we have a valid authentication cookie
func (p *CloudflareAccessProxy) isAuthenticated() bool {
	return p.cfCookie != "" && time.Now().Unix() < p.cookieExpires
}

// ensureAuthenticated ensures we have valid authentication
func (p *CloudflareAccessProxy) ensureAuthenticated() bool {
	if p.isAuthenticated() {
		return true
	}

	// Try cloudflared authentication first
	if p.authenticateWithCloudflared() {
		return true
	}

	// Try to get cookie from browser
	if p.extractCFCookieFromBrowser() {
		return true
	}

	// Fall back to manual authentication
	return p.authenticateManual()
}

// proxyHandler handles HTTP requests
type proxyHandler struct {
	proxy *CloudflareAccessProxy
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(proxy *CloudflareAccessProxy) *proxyHandler {
	return &proxyHandler{proxy: proxy}
}

// ServeHTTP implements http.Handler interface
func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Ensure we're authenticated
	if !h.proxy.ensureAuthenticated() {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Build target URL
	targetURL := fmt.Sprintf("https://%s%s", h.proxy.targetDomain, r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Read request body if present
	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Create a new request
	var req *http.Request
	if len(bodyBytes) > 0 {
		req, err = http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
	} else {
		req, err = http.NewRequest(r.Method, targetURL, nil)
	}
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request, excluding problematic ones
	for name, values := range r.Header {
		// Skip host and accept-encoding headers
		if strings.ToLower(name) == "host" || strings.ToLower(name) == "accept-encoding" {
			continue
		}
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	// Fix OCI manifest support for registry requests
	if strings.HasPrefix(r.URL.Path, "/v2/") && req.Header.Get("Accept") != "" {
		accept := req.Header.Get("Accept")
		ociTypes := []string{
			"application/vnd.oci.image.manifest.v1+json",
			"application/vnd.oci.image.index.v1+json",
		}
		for _, ociType := range ociTypes {
			if !strings.Contains(accept, ociType) {
				accept += ", " + ociType
			}
		}
		req.Header.Set("Accept", accept)
	}

	// Add Cloudflare Access cookie
	cookieValue := fmt.Sprintf("CF_Authorization=%s", h.proxy.cfCookie)
	if existingCookie := req.Header.Get("Cookie"); existingCookie != "" {
		cookieValue = fmt.Sprintf("%s; %s", existingCookie, cookieValue)
	}
	req.Header.Set("Cookie", cookieValue)

	// Make the proxied request
	resp, err := h.proxy.session.Do(req)
	if err != nil {
		h.proxy.logger.Errorf("❌ Proxy error: %v", err)
		http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy response headers first (before WriteHeader)
	for name, values := range resp.Header {
		// Skip headers that could cause issues
		if strings.ToLower(name) == "transfer-encoding" || strings.ToLower(name) == "connection" {
			continue
		}
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Copy response status
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if r.Method != "HEAD" {
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			h.proxy.logger.Errorf("❌ Failed to copy response body: %v", err)
		}
	}
	
	// Log the request (similar to Python's log_message)
	h.proxy.logger.Infof("🔄 %s %s %d", r.Method, r.URL.Path, resp.StatusCode)
}

// main function
func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: cf-proxy <target_domain> [--port <port>]")
		os.Exit(1)
	}

	domain := os.Args[1]
	port := 8081

	if len(os.Args) > 3 && os.Args[2] == "--port" {
		p, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("Invalid port number")
			os.Exit(1)
		}
		port = p
	}

	fmt.Printf("🚀 Starting Cloudflare Access Proxy\n")
	fmt.Printf("📡 Target: %s\n", domain)
	fmt.Printf("🔗 Local proxy: http://localhost:%d\n", port)

	// Create proxy instance
	proxy := NewCloudflareAccessProxy(domain, port)

	// Authenticate upfront before starting server
	fmt.Println("\n🔐 Authentication required before starting proxy...")
	if !proxy.ensureAuthenticated() {
		fmt.Println("❌ Authentication failed. Exiting.")
		os.Exit(1)
	}
	fmt.Println("✅ Authentication successful!")

	// Create HTTP server
	handler := NewProxyHandler(proxy)
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: handler,
	}

	fmt.Printf("\n🚀 Proxy running on http://localhost:%d\n", port)
	fmt.Printf("💡 Configure your client to use localhost:%d instead of %s\n", port, domain)
	fmt.Println("🛑 Press Ctrl+C to stop")

	// Start server
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("❌ Server error: %v\n", err)
		os.Exit(1)
	}
}