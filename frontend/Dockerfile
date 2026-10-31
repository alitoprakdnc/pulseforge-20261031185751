# syntax=docker/dockerfile:1.4

############################
# Builder Stage
############################
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Install only production dependencies first (helps with layer caching)
COPY package.json ./
# Prefer lockfile if present
COPY package-lock.json .npmrc ./
RUN if [ -f package-lock.json ]; then \
        npm ci --silent; \
    else \
        npm install --silent; \
    fi && npm cache clean --force

# Copy the rest of the source code
COPY . .

# Build the Next.js app and export it as a static site
# Assumes scripts: "build": "next build" and "export": "next export"
RUN npm run build && npm run export

# Remove dev dependencies to keep the image lean
RUN npm prune --production

############################
# Production Stage (nginx)
############################
FROM nginx:alpine

# Labels for maintainability
LABEL maintainer="PulseForge Team <dev@pulseforge.io>"
LABEL description="PulseForge frontend – static Next.js site served via nginx"
LABEL org.opencontainers.image.source="https://github.com/your-org/pulseforge"

# Copy the static export from the builder stage
COPY --from=builder /app/out /usr/share/nginx/html

# Optional: copy a custom nginx configuration if you need to tweak defaults
# The file should exist in the repository root as `nginx.conf`
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose the HTTP port
EXPOSE 80

# Basic health‑check – ensure nginx returns a 200 on the root path
HEALTHCHECK --interval=30s --timeout=5s \
  CMD wget -qO- http://localhost/ || exit 1

# Run nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]