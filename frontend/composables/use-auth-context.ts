import type { CookieRef, NuxtApp } from "nuxt/app";
import type { Ref } from "vue";
import type { PublicApi } from "~~/lib/api/public";
import type { UserSelfOut } from "~~/lib/api/types/data-contracts";
import type { UserClient } from "~~/lib/api/user";

export interface IAuthContext {
  get token(): boolean | null;
  get attachmentToken(): string | null;

  /**
   * The current user object for the session. This is undefined if the session is not authorized.
   */
  user?: UserSelfOut;

  /**
   * Returns true if the session is authorized.
   */
  isAuthorized(): boolean;

  /**
   * Invalidates the session by removing the token and the expiresAt.
   */
  invalidateSession(): void;

  /**
   * Logs out the user and calls the invalidateSession method.
   */
  logout(api: UserClient): ReturnType<UserClient["user"]["logout"]>;

  /**
   * Logs in the user and sets the authorization context via cookies
   */
  login(api: PublicApi, email: string, password: string, stayLoggedIn: boolean): ReturnType<PublicApi["login"]>;
}

// One AuthContext per Nuxt app instance. A module-level singleton would leak
// cookie refs between requests during SSR (cross-request state pollution).
class AuthContext implements IAuthContext {
  private static readonly cookieTokenKey = "hb.auth.session";
  private static readonly cookieAttachmentTokenKey = "hb.auth.attachment_token";

  private _user: Ref<UserSelfOut | undefined>;
  private _token: CookieRef<string | null>;
  private _attachmentToken: CookieRef<string | null>;

  get user() {
    return this._user.value;
  }

  set user(user: UserSelfOut | undefined) {
    this._user.value = user;
  }

  get token() {
    // @ts-expect-error sometimes it's a boolean I guess?
    return this._token.value === "true" || this._token.value === true;
  }

  get attachmentToken() {
    return this._attachmentToken.value;
  }

  constructor(private readonly nuxtApp: NuxtApp) {
    // cast needed because the auto-imported global Ref and the "vue" Ref types
    // don't unify in this codebase
    this._user = useState<UserSelfOut | undefined>("auth.user", () => undefined) as unknown as Ref<
      UserSelfOut | undefined
    >;
    this._token = useCookie(AuthContext.cookieTokenKey);
    this._attachmentToken = useCookie(AuthContext.cookieAttachmentTokenKey);
  }

  isExpired() {
    return !this.token;
  }

  isAuthorized() {
    console.debug("isAuthorized", this.token);
    return this.token;
  }

  invalidateSession() {
    this.user = undefined;

    // Delete the cookies
    this._token.value = null;
    this._attachmentToken.value = null;
    console.log("Session invalidated");
  }

  async login(api: PublicApi, email: string, password: string, stayLoggedIn: boolean) {
    const r = await api.login(email, password, stayLoggedIn);

    if (!r.error) {
      const expiresAt = new Date(r.data.expiresAt);
      // useCookie needs the Nuxt context, which is lost after the await above
      await this.nuxtApp.runWithContext(() => {
        this._token = useCookie(AuthContext.cookieTokenKey);
        this._attachmentToken = useCookie(AuthContext.cookieAttachmentTokenKey, {
          expires: expiresAt,
        });
        this._attachmentToken.value = r.data.attachmentToken;
      });
    }

    return r;
  }

  async logout(api: UserClient) {
    const r = await api.user.logout();

    if (!r.error) {
      this.invalidateSession();
    }

    return r;
  }
}

export function useAuthContext(): IAuthContext {
  const nuxtApp = useNuxtApp() as NuxtApp & { _authContext?: AuthContext };
  nuxtApp._authContext ??= new AuthContext(nuxtApp);
  return nuxtApp._authContext;
}
