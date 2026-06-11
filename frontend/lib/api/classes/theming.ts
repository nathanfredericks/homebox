import { BaseAPI, route } from "../base";
import type { ThemeCreate, ThemeOut, ThemeUpdate, ThemingSettings } from "../types/data-contracts";

export type ThemeAssetKind = "nav-logo" | "sidebar-logo" | "login-icon";

/**
 * Admin-managed instance themes (custom palettes + whitelabel branding) and
 * the site-wide active theme pointer.
 */
export class ThemingApi extends BaseAPI {
  getAll() {
    return this.http.get<ThemeOut[]>({ url: route("/themes") });
  }

  get(id: string) {
    return this.http.get<ThemeOut>({ url: route(`/themes/${id}`) });
  }

  create(data: ThemeCreate) {
    return this.http.post<ThemeCreate, ThemeOut>({ url: route("/themes"), body: data });
  }

  update(id: string, data: ThemeUpdate) {
    return this.http.put<ThemeUpdate, ThemeOut>({ url: route(`/themes/${id}`), body: data });
  }

  delete(id: string) {
    return this.http.delete<void>({ url: route(`/themes/${id}`) });
  }

  uploadAsset(id: string, kind: ThemeAssetKind, file: File) {
    const formData = new FormData();
    formData.append("file", file);

    return this.http.post<FormData, ThemeOut>({
      url: route(`/themes/${id}/assets/${kind}`),
      data: formData,
    });
  }

  deleteAsset(id: string, kind: ThemeAssetKind) {
    return this.http.delete<ThemeOut>({ url: route(`/themes/${id}/assets/${kind}`) });
  }

  /** Authenticated preview URL for the editor (works for inactive themes). */
  assetUrl(id: string, kind: ThemeAssetKind, version?: string | number) {
    const url = this.authURL(`/themes/${id}/assets/${kind}`);
    if (!version) {
      return url;
    }
    return url + (url.includes("?") ? "&" : "?") + "v=" + version;
  }

  getActive() {
    return this.http.get<ThemingSettings>({ url: route("/theming/active") });
  }

  setActive(active: string) {
    return this.http.put<ThemingSettings, ThemingSettings>({ url: route("/theming/active"), body: { active } });
  }
}
