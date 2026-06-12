import { BaseAPI, route } from "../base";

/**
 * Instance-wide label sheet layout (Administration → Settings → Label maker).
 * Mirrors config.LabelMakerConf on the backend. Readable by any authenticated
 * user so the browser-side label renderer can lay out sheets.
 */
export type LabelLayout = {
  baseUrl: string;
  measure: string;
  cardWidth: number;
  cardHeight: number;
  pageWidth: number;
  pageHeight: number;
  pageTopPadding: number;
  pageBottomPadding: number;
  pageLeftPadding: number;
  pageRightPadding: number;
  // Google Font family names; "default" keeps the built-in font stacks.
  sansFont: string;
  monoFont: string;
  bordered: boolean;
  printLocationRow: boolean;
  labelPerQuantity: boolean;
};

export class LabelMakerApi extends BaseAPI {
  settings() {
    return this.http.get<LabelLayout>({ url: route("/labelmaker/settings") });
  }
}
