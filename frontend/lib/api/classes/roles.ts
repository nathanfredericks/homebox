import { BaseAPI, route } from "../base";
import type { RoleCreate, RoleOut, RoleUpdate } from "../types/data-contracts";

/**
 * Roles are presented as "Groups" in the UI: named bundles of granular
 * permissions that users are assigned to.
 */
export class RolesApi extends BaseAPI {
  getAll() {
    return this.http.get<RoleOut[]>({ url: route("/roles") });
  }

  get(id: string) {
    return this.http.get<RoleOut>({ url: route(`/roles/${id}`) });
  }

  create(data: RoleCreate) {
    return this.http.post<RoleCreate, RoleOut>({ url: route("/roles"), body: data });
  }

  update(id: string, data: RoleUpdate) {
    return this.http.put<RoleUpdate, RoleOut>({ url: route(`/roles/${id}`), body: data });
  }

  delete(id: string) {
    return this.http.delete<void>({ url: route(`/roles/${id}`) });
  }
}
