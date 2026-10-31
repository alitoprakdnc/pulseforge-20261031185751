/** @type {import('next').NextConfig} */
const nextConfig = {
  // Enable React strict mode and SWC minification for production builds
  reactStrictMode: true,
  swcMinify: true,

  // ---------------------------------------------------------------------------
  // Image Optimization
  // ---------------------------------------------------------------------------
  // No external image domains are required for the current UI, but the config
  // is kept extensible for future use (e.g., avatars from GitHub).
  images: {
    domains: [], // add allowed domains here, e.g. ['avatars.githubusercontent.com']
    remotePatterns: [],
  },

  // ---------------------------------------------------------------------------
  // Environment variables
  // ---------------------------------------------------------------------------
  // NEXT_PUBLIC_BACKEND_URL is used by the client to call the Go API.
  // It defaults to the local development address when not provided.
  env: {
    NEXT_PUBLIC_BACKEND_URL:
      process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080',
  },

  // ---------------------------------------------------------------------------
  // API Proxy (rewrites)
  // ---------------------------------------------------------------------------
  // All calls to `/api/*` from the frontend are proxied to the backend service.
  // This keeps the same origin for the browser, simplifying CORS and cookie
  // handling (HttpOnly JWT cookie).
  async rewrites() {
    const backend = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
    return [
      {
        source: '/api/:path*',
        destination: `${backend}/api/:path*`,
      },
    ];
  },

  // ---------------------------------------------------------------------------
  // Security Headers
  // ---------------------------------------------------------------------------
  // Apply a sensible set of security headers to every response.
  async headers() {
    const backend = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
    return [
      {
        // Apply to all routes
        source: '/(.*)',
        headers: [
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
          {
            key: 'Permissions-Policy',
            value: 'camera=(), microphone=(), geolocation=()',
          },
          {
            // Content Security Policy tailored for the app
            key: 'Content-Security-Policy',
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-eval' 'unsafe-inline'",
              "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
              "img-src 'self' data:",
              "font-src 'self' https://fonts.gstatic.com",
              `connect-src 'self' ${backend}`,
            ].join('; '),
          },
        ],
      },
    ];
  },

  // ---------------------------------------------------------------------------
  // Future & Experimental Flags
  // ---------------------------------------------------------------------------
  // Keep webpack 5 (default) and enable future features if needed.
  future: {
    webpack5: true,
  },
};

module.exports = nextConfig;