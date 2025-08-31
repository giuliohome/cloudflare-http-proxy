FROM python:3.11-slim

LABEL maintainer="Giulio"
LABEL description="Cloudflare Access Cookie Proxy - Transparent authentication proxy for Cloudflare Access protected services"

# Install required packages
RUN pip install --no-cache-dir requests

# Create app directory
WORKDIR /app

# Copy the proxy script
COPY cf_proxy.py .

# Make the script executable
RUN chmod +x cf_proxy.py

# Expose the default port
EXPOSE 8081

# Create a non-root user for security
RUN useradd --create-home --shell /bin/bash proxy && \
    chown -R proxy:proxy /app
USER proxy

# Default command
ENTRYPOINT ["python3", "cf_proxy.py"]
CMD ["--help"]