import { BaseAPI, route } from "../base";
import type { CurrenciesCurrency, Group, GroupUpdate } from "../types/data-contracts";

export class GroupApi extends BaseAPI {
  /**
   * Update group name and currency.
   */
  update(data: GroupUpdate, groupId?: string) {
    const headers = groupId
      ? {
          "X-Tenant": groupId,
        }
      : undefined;
    return this.http.put<GroupUpdate, Group>({
      url: route(`/groups`),
      headers,
      body: data,
    });
  }

  /**
   * Get a group by ID, if no ID is provided, get the current group.
   */
  get(groupId?: string) {
    const headers = groupId
      ? {
          "X-Tenant": groupId,
        }
      : undefined;
    return this.http.get<Group>({
      url: route(`/groups`),
      headers,
    });
  }

  /**
   * Get all collections the user can access.
   */
  getAll() {
    return this.http.get<Group[]>({
      url: route("/groups/all"),
    });
  }

  /**
   * Create a new group with the given name.
   */
  create(name: string) {
    return this.http.post<
      {
        name: string;
      },
      Group
    >({
      url: route("/groups"),
      body: { name },
    });
  }

  /**
   * Delete a group by ID, if no ID is provided, delete the current group.
   */
  delete(groupId?: string) {
    const headers = groupId
      ? {
          "X-Tenant": groupId,
        }
      : undefined;
    return this.http.delete<void>({
      url: route(`/groups`),
      headers,
    });
  }

  /**
   * Get all currencies.
   */
  currencies() {
    return this.http.get<CurrenciesCurrency[]>({
      url: route("/currencies"),
    });
  }
}
