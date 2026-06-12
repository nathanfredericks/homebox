import { BaseAPI, route } from "../base";
import type { AdminSettingsOut } from "../types/data-contracts";

export type SettingsSection =
  | "options"
  | "thumbnail"
  | "mailer"
  | "barcode"
  | "labelmaker"
  | "notifier"
  | "algolia"
  | "ai";

/**
 * Site-wide settings (Administration → Settings). Values layer database
 * overrides on top of environment variables; secret fields read back as the
 * "[REDACTED]" sentinel and keep their stored value when the sentinel is
 * sent back unchanged.
 */
export class AdminSettingsApi extends BaseAPI {
  get() {
    return this.http.get<AdminSettingsOut>({ url: route("/admin/settings") });
  }

  /** Persist a sparse override document for one section. */
  updateSection(section: SettingsSection, body: Record<string, unknown>) {
    return this.http.put<Record<string, unknown>, AdminSettingsOut>({
      url: route(`/admin/settings/${section}`),
      body,
    });
  }

  /** Drop a section's database override, restoring environment values. */
  resetSection(section: SettingsSection) {
    return this.http.delete<AdminSettingsOut>({
      url: route(`/admin/settings/${section}`),
    });
  }

  /** Kick an asynchronous full Algolia reindex. */
  algoliaReindex() {
    return this.http.post<Record<string, never>, { status: string }>({
      url: route("/admin/settings/algolia/reindex"),
      body: {},
    });
  }
}
