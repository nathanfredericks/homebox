import type { Component } from "vue";
import { computed } from "vue";
import MdiAccountMultiple from "~icons/mdi/account-multiple";
import MdiShieldAccount from "~icons/mdi/shield-account";
import { usePermissions } from "~/composables/use-permissions";

export type AdminSection = {
  id: string;
  labelKey: string;
  to: string;
  icon: Component;
};

/**
 * Ordered list of Administration sections visible to the current user.
 * Single source of truth for the sidebar submenu, the /admin landing
 * redirect, and the shell header; sections the user cannot view do not
 * exist for them.
 */
export function useAdminSections() {
  const { can } = usePermissions();

  const sections = computed<AdminSection[]>(() =>
    [
      {
        id: "users",
        labelKey: "admin.tabs.users",
        to: "/admin/users",
        icon: MdiAccountMultiple,
        visible: can("users", "view"),
      },
      {
        id: "groups",
        labelKey: "admin.tabs.groups",
        to: "/admin/groups",
        icon: MdiShieldAccount,
        visible: can("roles", "view"),
      },
    ]
      .filter(section => section.visible)
      .map(({ visible: _visible, ...section }) => section)
  );

  return { sections };
}
