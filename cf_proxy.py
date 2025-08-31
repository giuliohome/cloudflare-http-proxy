#!/usr/bin/env python3
"""
Cloudflare Access Cookie Proxy
Sits between client apps and Cloudflare Access protected services
Handles cookie authentication transparently
"""

import requests
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
import urllib.parse
import json
import time
from typing import Optional

class CloudflareAccessProxy:
    def __init__(self, target_domain: str, local_port: int = 8081):
        self.target_domain = target_domain
        self.local_port = local_port
        self.session = requests.Session()
        self.cf_cookie: Optional[str] = None
        self.cookie_expires = 0

    def extract_cf_cookie_from_browser(self) -> bool:
        """
        Attempts to extract CF_Authorization cookie from browser
        You'd need to implement browser cookie extraction here
        For now, returns False to trigger manual auth
        """
        # TODO: Implement browser cookie extraction
        # Could use browser_cookie3 library or similar
        return False

    def authenticate_manual(self) -> bool:
        """
        Manual authentication - user provides cookie
        """
        print(f"\n🔐 Authentication required for {self.target_domain}")
        print("1. Open your browser and go to:")
        print(f"   https://{self.target_domain}")
        print("2. Complete Cloudflare authentication")
        print("3. Open browser dev tools → Application → Cookies")
        print(f"4. Find 'CF_Authorization' cookie for {self.target_domain}")
        print("5. Copy the cookie value and paste it here:")

        cookie_value = input("\nCF_Authorization cookie: ").strip()

        if cookie_value:
            self.cf_cookie = cookie_value
            # Set expiration to 23 hours from now (typical CF Access session)
            self.cookie_expires = time.time() + (23 * 3600)
            return True
        return False

    def is_authenticated(self) -> bool:
        """Check if we have a valid authentication cookie"""
        return (self.cf_cookie is not None and
                time.time() < self.cookie_expires)

    def ensure_authenticated(self) -> bool:
        """Ensure we have valid authentication"""
        if self.is_authenticated():
            return True

        # Try to get cookie from browser first
        if self.extract_cf_cookie_from_browser():
            return True

        # Fall back to manual authentication
        return self.authenticate_manual()

class ProxyHandler(BaseHTTPRequestHandler):
    def __init__(self, proxy_instance, *args, **kwargs):
        self.proxy = proxy_instance
        super().__init__(*args, **kwargs)

    def do_GET(self):
        self._proxy_request('GET')

    def do_POST(self):
        self._proxy_request('POST')

    def do_PUT(self):
        self._proxy_request('PUT')

    def do_DELETE(self):
        self._proxy_request('DELETE')

    def do_HEAD(self):
        self._proxy_request('HEAD')

    def _proxy_request(self, method: str):
        """Proxy the request to the target domain with CF authentication"""

        # Ensure we're authenticated
        if not self.proxy.ensure_authenticated():
            self.send_error(401, "Authentication failed")
            return

        # Build target URL
        target_url = f"https://{self.proxy.target_domain}{self.path}"

        # Prepare headers
        headers = {}
        for header, value in self.headers.items():
            # Skip host header as we're changing the target
            # Also skip accept-encoding to avoid compression issues
            if header.lower() not in ['host', 'accept-encoding']:
                headers[header] = value

        # Fix OCI manifest support for registry requests
        if self.path.startswith('/v2/') and ('Accept' in headers):
            # Always add OCI manifest support for registry API calls
            oci_types = [
                'application/vnd.oci.image.manifest.v1+json',
                'application/vnd.oci.image.index.v1+json'
            ]
            current_accept = headers['Accept']
            for oci_type in oci_types:
                if oci_type not in current_accept:
                    headers['Accept'] += f', {oci_type}'

        # Add Cloudflare Access cookie
        if 'Cookie' in headers:
            headers['Cookie'] += f"; CF_Authorization={self.proxy.cf_cookie}"
        else:
            headers['Cookie'] = f"CF_Authorization={self.proxy.cf_cookie}"

        try:
            # Read request body if present
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length) if content_length > 0 else None

            # Make the proxied request
            response = self.proxy.session.request(
                method=method,
                url=target_url,
                headers=headers,
                data=body,
                stream=True,
                allow_redirects=False
            )

            # Send response status
            self.send_response(response.status_code)

            # Forward response headers
            for header, value in response.headers.items():
                # Skip headers that could cause issues
                if header.lower() not in ['transfer-encoding', 'connection']:
                    self.send_header(header, value)
            self.end_headers()

            # Forward response body
            if method != 'HEAD':
                for chunk in response.iter_content(chunk_size=8192):
                    if chunk:
                        self.wfile.write(chunk)

        except Exception as e:
            print(f"❌ Proxy error: {e}")
            self.send_error(500, f"Proxy error: {str(e)}")

    def log_message(self, format, *args):
        """Custom logging"""
        print(f"🔄 {format % args}")

def create_handler(proxy_instance):
    """Create handler class with proxy instance"""
    def handler(*args, **kwargs):
        return ProxyHandler(proxy_instance, *args, **kwargs)
    return handler

def main():
    import argparse

    parser = argparse.ArgumentParser(description='Cloudflare Access Cookie Proxy')
    parser.add_argument('domain', help='Target domain (e.g., registry.yourdomain.com)')
    parser.add_argument('--port', type=int, default=8081, help='Local proxy port')

    args = parser.parse_args()

    print(f"🚀 Starting Cloudflare Access Proxy")
    print(f"📡 Target: {args.domain}")
    print(f"🔗 Local proxy: http://localhost:{args.port}")

    # Create proxy instance
    proxy = CloudflareAccessProxy(args.domain, args.port)

    # Authenticate upfront before starting server
    print("\n🔐 Authentication required before starting proxy...")
    if not proxy.ensure_authenticated():
        print("❌ Authentication failed. Exiting.")
        return

    print("✅ Authentication successful!")

    # Create HTTP server
    server = HTTPServer(('localhost', args.port), create_handler(proxy))

    print(f"\n🚀 Proxy running on http://localhost:{args.port}")
    print(f"💡 Configure your client to use localhost:{args.port} instead of {args.domain}")
    print("🛑 Press Ctrl+C to stop")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n👋 Shutting down proxy...")
        server.shutdown()

if __name__ == "__main__":
    main()
