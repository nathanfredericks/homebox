import { PublicApi } from "~~/lib/api/public";
import { UserClient } from "~~/lib/api/user";
import { Requests } from "~~/lib/requests";

export type Observer = {
  handler: (r: Response, req?: RequestInit) => void;
};

export type RemoveObserver = () => void;

const observers: Record<string, Observer> = {};

export function defineObserver(key: string, observer: Observer): RemoveObserver {
  observers[key] = observer;

  return () => {
    // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
    delete observers[key];
  };
}

function logger(r: Response) {
  console.log(`${r.status}   ${r.url}   ${r.statusText}`);
}

// In the browser requests stay relative (same origin, routed by the reverse
// proxy). During SSR there is no origin, so requests go straight to the Go API.
function apiBaseUrl(): string {
  if (import.meta.server) {
    return useRuntimeConfig().apiHost;
  }
  return "";
}

// During SSR the browser's cookies (incl. the HttpOnly auth token) must be
// forwarded to the Go API; the browser sends them itself.
function forwardedHeaders(): Record<string, string> {
  if (import.meta.client) {
    return {};
  }

  const headers: Record<string, string> = {};
  const { cookie } = useRequestHeaders(["cookie"]);
  if (cookie) {
    headers.Cookie = cookie;
  }
  return headers;
}

export function usePublicApi(): PublicApi {
  const requests = new Requests(apiBaseUrl(), "", forwardedHeaders());
  return new PublicApi(requests);
}

export function useUserApi(): UserClient {
  const authCtx = useAuthContext();
  const prefs = useViewPreferences();

  const headers: Record<string, string> = forwardedHeaders();
  if (prefs?.value?.collectionId) {
    headers["X-Tenant"] = prefs.value.collectionId;
  }

  const requests = new Requests(apiBaseUrl(), "", headers);
  requests.addResponseInterceptor(logger);
  requests.addResponseInterceptor(async r => {
    if (r.status === 401) {
      console.error("unauthorized request, invalidating session");
      authCtx.invalidateSession();
      if (import.meta.client) {
        navigateTo("/");
      }
    }

    if (r.status === 403 && import.meta.client) {
      try {
        const contentType = r.headers.get("Content-Type") ?? "";
        if (!contentType.startsWith("application/json")) {
          return;
        }

        const body = (await r.json().catch(() => null)) as { error?: string } | null;

        if (body?.error === "user does not have access to the requested tenant") {
          console.log("user does not have access to the requested tenant");
          if (window.location.pathname == "/") {
            // do nothing
            console.log("at root path, ignoring collectionId to prevent infinite redirect loop");
          } else if (!prefs?.value?.collectionId) {
            console.log("no collectionId set, ignoring");
          } else {
            console.log("clearing collectionId");
            prefs.value.collectionId = null;
          }
        }
      } catch {
        // ignore parsing errors to avoid breaking the interceptor chain
        console.log("failed to parse 403 response body");
      }
    }
  });

  for (const [_, observer] of Object.entries(observers)) {
    requests.addResponseInterceptor(observer.handler);
  }

  return new UserClient(requests, authCtx.attachmentToken || "");
}
