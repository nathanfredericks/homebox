import type { Component } from "vue";
import { computed } from "vue";
import MdiCog from "~icons/mdi/cog";
import MdiShieldAccount from "~icons/mdi/shield-account";
import MdiBell from "~icons/mdi/bell";
import MdiShape from "~icons/mdi/shape";
import MdiPrinter from "~icons/mdi/printer";
import MdiWrench from "~icons/mdi/wrench";
import { usePermissions } from "~/composables/use-permissions";

export type CollectionSection = {
  id: string;
  labelKey: string;
  to: string;
  icon: Component;
};

/**
 * Ordered list of Collection Settings sections visible to the current user.
 * Single source of truth for the sidebar submenu, the /collection landing
 * redirect, and the shell header; sections the user cannot view do not
 * exist for them.
 */
export function useCollectionSections() {
  const { can } = usePermissions();

  const sections = computed<CollectionSection[]>(() =>
    [
      {
        id: "settings",
        labelKey: "collection.tabs.general",
        to: "/collection/settings",
        icon: MdiCog,
        visible: can("collection_settings", "view"),
      },
      {
        id: "access",
        labelKey: "collection.tabs.access",
        to: "/collection/access",
        icon: MdiShieldAccount,
        visible: can("roles", "view"),
      },
      {
        id: "notifiers",
        labelKey: "collection.tabs.notifiers",
        to: "/collection/notifiers",
        icon: MdiBell,
        visible: can("notifiers", "view"),
      },
      {
        id: "entity-types",
        labelKey: "collection.tabs.entity_types",
        to: "/collection/entity-types",
        icon: MdiShape,
        visible: can("entity_types", "view"),
      },
      {
        // The label generator lived under Tools, so it keeps that permission.
        id: "labels",
        labelKey: "collection.tabs.labels",
        to: "/collection/labels",
        icon: MdiPrinter,
        visible: can("tools", "view"),
      },
      {
        id: "tools",
        labelKey: "collection.tabs.tools",
        to: "/collection/tools",
        icon: MdiWrench,
        visible: can("tools", "view"),
      },
    ]
      .filter(section => section.visible)
      .map(({ visible: _visible, ...section }) => section)
  );

  return { sections };
}
