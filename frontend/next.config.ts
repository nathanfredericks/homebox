import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
  async rewrites() {
    // Dev-only: proxy HTTP API calls to the Go backend. In production the
    // reverse proxy (Caddy) routes /api/* and the WebSocket upgrade; this
    // rewrite is never used there. WebSocket connections dial the Go server
    // directly in dev (host port swap in use-server-events), so only HTTP is
    // proxied here.
    if (process.env.NODE_ENV !== "development") {
      return [];
    }
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:7745/api/:path*",
      },
    ];
  },
};

export default nextConfig;
