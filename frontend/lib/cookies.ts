/**
 * Minimal document.cookie helpers replacing Nuxt's `useCookie`.
 *
 * Homebox uses three client-readable cookies (the real session token is an
 * HttpOnly cookie the browser sends automatically and is never touched here):
 *   - `hb.auth.session`           — a boolean flag set to the string "true"
 *                                   when a session exists. The Go backend
 *                                   sometimes serializes it as a bare boolean,
 *                                   so reads normalize both forms.
 *   - `hb.auth.attachment_token`  — used to build attachment/QR URLs and to
 *                                   authenticate the WebSocket subprotocol.
 *
 * Nuxt's `useCookie` defaults to `path=/`; we match that so existing cookies
 * set by the Go backend / prior sessions keep resolving.
 */

export const AUTH_SESSION_KEY = "hb.auth.session";
export const AUTH_ATTACHMENT_TOKEN_KEY = "hb.auth.attachment_token";

export interface CookieSetOptions {
  /** Absolute expiry. Omit for a session cookie. */
  expires?: Date;
  /** Max-Age in seconds (takes precedence over `expires` if both are given). */
  maxAge?: number;
  path?: string;
  sameSite?: "strict" | "lax" | "none";
  secure?: boolean;
}

/** Read a raw cookie value, or `null` if absent or running on the server. */
export function getCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }
  const prefix = `${encodeURIComponent(name)}=`;
  const cookies = document.cookie ? document.cookie.split("; ") : [];
  for (const cookie of cookies) {
    if (cookie.startsWith(prefix)) {
      return decodeURIComponent(cookie.slice(prefix.length));
    }
  }
  return null;
}

/** Write a cookie. Defaults to `path=/` to match Nuxt's `useCookie`. */
export function setCookie(name: string, value: string, options: CookieSetOptions = {}): void {
  if (typeof document === "undefined") {
    return;
  }
  const { expires, maxAge, path = "/", sameSite, secure } = options;
  let cookie = `${encodeURIComponent(name)}=${encodeURIComponent(value)}; path=${path}`;
  if (typeof maxAge === "number") {
    cookie += `; max-age=${Math.floor(maxAge)}`;
  } else if (expires) {
    cookie += `; expires=${expires.toUTCString()}`;
  }
  if (sameSite) {
    cookie += `; samesite=${sameSite}`;
  }
  if (secure) {
    cookie += "; secure";
  }
  document.cookie = cookie;
}

/** Remove a cookie by expiring it in the past (same path as it was written). */
export function deleteCookie(name: string, path = "/"): void {
  if (typeof document === "undefined") {
    return;
  }
  document.cookie = `${encodeURIComponent(name)}=; path=${path}; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
}

/**
 * Read the `hb.auth.session` flag, normalizing the string "true" and a bare
 * boolean `true` (the backend has been observed to serialize both).
 */
export function hasAuthSession(): boolean {
  const value = getCookie(AUTH_SESSION_KEY);
  return value === "true";
}

/** Read the attachment token used for attachment/QR URLs and WebSocket auth. */
export function getAttachmentToken(): string | null {
  return getCookie(AUTH_ATTACHMENT_TOKEN_KEY);
}

/** Persist the attachment token with an optional absolute expiry. */
export function setAttachmentToken(token: string, expires?: Date): void {
  setCookie(AUTH_ATTACHMENT_TOKEN_KEY, token, { expires });
}

/** Clear both client-readable auth cookies on logout / session invalidation. */
export function clearAuthCookies(): void {
  deleteCookie(AUTH_SESSION_KEY);
  deleteCookie(AUTH_ATTACHMENT_TOKEN_KEY);
}
