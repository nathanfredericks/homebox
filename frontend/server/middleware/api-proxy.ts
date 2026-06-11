// Dev-only /api proxy to the Go backend. nitro.devProxy (httpxy) leaves
// requests hanging forever when the backend is down; proxyRequest fails fast
// so a dead backend yields an immediate 502 instead of a stuck navigation.
// In production a reverse proxy routes /api before it ever reaches Nitro.
export default defineEventHandler(async event => {
  if (!import.meta.dev) {
    return;
  }

  if (!event.path.startsWith("/api/")) {
    return;
  }

  const apiHost = useRuntimeConfig().apiHost;

  try {
    return await proxyRequest(event, apiHost + event.path);
  } catch {
    throw createError({ statusCode: 502, statusMessage: "Bad Gateway" });
  }
});
