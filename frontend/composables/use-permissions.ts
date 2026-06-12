import { computed } from "vue";
import { useAuthContext } from "~/composables/use-auth-context";
import { useCollections } from "~/composables/use-collections";

export type PermSection =
  | "items"
  | "locations"
  | "tags"
  | "templates"
  | "maintenance"
  | "statistics"
  | "ai"
  | "collection_settings"
  | "entity_types"
  | "notifiers"
  | "tools"
  | "users"
  | "roles"
  | "collections"
  | "site_settings"
  | "theming";

export type PermAction = "view" | "create" | "edit" | "delete";

const SITE_SECTIONS: PermSection[] = ["users", "roles", "collections", "site_settings", "theming"];

/**
 * Permission gating for the UI. Anything the user cannot do simply does not
 * exist for them: callers must remove elements with v-if (never disable).
 * The backend remains the authority; this only mirrors the grant list that
 * ships on /users/self.
 */
export function usePermissions() {
  const auth = useAuthContext();
  const { selectedId } = useCollections();

  const isSuperAdmin = computed(() => auth.user?.isSuperAdmin === true);

  /**
   * Returns true when the user can perform the action on the section. For
   * collection-scoped sections the check targets the given collection (or
   * the currently selected one); a null-scoped grant covers all collections.
   */
  const can = (section: PermSection, action: PermAction, collectionId?: string | null): boolean => {
    if (auth.user?.isSuperAdmin) return true;

    const grants = auth.user?.permissions;
    if (!grants?.length) return false;

    const siteScoped = SITE_SECTIONS.includes(section);
    const target = siteScoped ? null : (collectionId ?? selectedId.value);

    return grants.some(g => {
      if (g.section !== section) return false;
      if (!g[action]) return false;
      // Null collectionId on a grant = all collections / site scope.
      return g.collectionId == null || (target != null && g.collectionId === target);
    });
  };

  /**
   * Returns true when the user can perform the action on the section in ANY
   * collection. Used for route-level gating, where no specific collection is
   * in scope yet; precise per-collection checks happen in the page UI and on
   * the backend.
   */
  const canAny = (section: PermSection, action: PermAction): boolean => {
    if (auth.user?.isSuperAdmin) return true;
    const grants = auth.user?.permissions;
    if (!grants?.length) return false;
    return grants.some(g => g.section === section && g[action]);
  };

  /** True when any administration surface is visible to the user. */
  const canAdminArea = computed(
    () =>
      isSuperAdmin.value ||
      can("users", "view") ||
      can("roles", "view") ||
      can("collections", "create") ||
      can("site_settings", "view") ||
      can("theming", "view")
  );

  return { can, canAny, isSuperAdmin, canAdminArea };
}
