import { BaseAPI, route } from "../base";
import type { UserAdminCreate, UserAdminOut, UserAdminUpdate } from "../types/data-contracts";

/**
 * Site-level user administration. Users are created by admins only;
 * self-registration exists solely for first-time setup.
 */
export class AdminUsersApi extends BaseAPI {
  getAll() {
    return this.http.get<UserAdminOut[]>({ url: route("/users") });
  }

  create(data: UserAdminCreate) {
    return this.http.post<UserAdminCreate, UserAdminOut>({ url: route("/users"), body: data });
  }

  update(id: string, data: UserAdminUpdate) {
    return this.http.put<UserAdminUpdate, UserAdminOut>({ url: route(`/users/${id}`), body: data });
  }

  delete(id: string) {
    return this.http.delete<void>({ url: route(`/users/${id}`) });
  }
}
