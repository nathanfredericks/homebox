import { defineStore } from "pinia";
import type { ItemsApi } from "~~/lib/api/classes/items";
import type { EntitySummary, TreeItem } from "~~/lib/api/types/data-contracts";

// Pinia state is serialized into the SSR payload, so it must hold plain data
// only — no API client instances and no in-flight Promises. Dedupe promises
// live on the store instance outside of $state.
type RefreshTracker = {
  _refreshLocations?: Promise<unknown>;
  _refreshParents?: Promise<unknown>;
};

export const useLocationStore = defineStore("locations", {
  state: () => ({
    parents: null as EntitySummary[] | null,
    Locations: null as EntitySummary[] | null,
    tree: null as TreeItem[] | null,
  }),
  getters: {
    /**
     * locations represents the locations that are currently in the store. The store is
     * synched with the server by intercepting the API calls and updating on the
     * response
     */
    parentLocations(state): EntitySummary[] {
      return state.parents ?? [];
    },
    allLocations(state): EntitySummary[] {
      return state.Locations ?? [];
    },
  },
  actions: {
    async ensureLocationsFetched() {
      if (this.Locations !== null) {
        return;
      }

      const self = this as typeof this & RefreshTracker;
      self._refreshLocations ??= this.refreshChildren();
      try {
        await self._refreshLocations;
      } finally {
        self._refreshLocations = undefined;
      }
    },
    async ensureParentsFetched() {
      if (this.parents !== null) {
        return;
      }

      const self = this as typeof this & RefreshTracker;
      self._refreshParents ??= this.refreshParents();
      try {
        await self._refreshParents;
      } finally {
        self._refreshParents = undefined;
      }
    },
    async refreshParents(): ReturnType<ItemsApi["getLocations"]> {
      const result = await useUserApi().items.getLocations({ filterChildren: true });
      if (result.error) {
        return result;
      }

      this.parents = result.data;
      return result;
    },
    async refreshChildren(): ReturnType<ItemsApi["getLocations"]> {
      const result = await useUserApi().items.getLocations({ filterChildren: false });
      if (result.error) {
        return result;
      }

      this.Locations = result.data;
      return result;
    },
    async refreshTree(): ReturnType<ItemsApi["getTree"]> {
      const result = await useUserApi().items.getTree();
      if (result.error) {
        return result;
      }

      this.tree = result.data;
      return result;
    },
  },
});
