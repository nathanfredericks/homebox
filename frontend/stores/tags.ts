import { defineStore } from "pinia";
import type { TagOut, TagSummary } from "~~/lib/api/types/data-contracts";

export const useTagStore = defineStore("tags", {
  // Pinia state is serialized into the SSR payload, so it must hold plain
  // data only — no API client instances and no in-flight Promises.
  state: () => ({
    allTags: null as TagOut[] | null,
  }),
  getters: {
    /**
     * tags represents the tags that are currently in the store. The store is
     * synched with the server by intercepting the API calls and updating on the
     * response.
     */
    tags(state): TagOut[] {
      return state.allTags ?? [];
    },
  },
  actions: {
    async ensureAllTagsFetched() {
      if (this.allTags !== null) {
        return;
      }

      // dedupe promise lives outside $state so it is never serialized
      const self = this as typeof this & { _refreshAllTags?: Promise<unknown> };
      self._refreshAllTags ??= this.refresh();
      try {
        await self._refreshAllTags;
      } finally {
        self._refreshAllTags = undefined;
      }
    },
    async refresh() {
      const result = await useUserApi().tags.getAll();
      if (result.error) {
        return result;
      }

      this.allTags = result.data;
      return result;
    },
    getAncestors(tags: string[]) {
      if (this.allTags === null) {
        return [];
      }

      // recursively find all ancestors of all input tags
      const toCheck = [this.allTags.filter(t => tags.includes(t.id))];
      const ancestors: TagOut[] = [];

      while (toCheck.length > 0) {
        const next = toCheck.pop();
        if (next === undefined) {
          break;
        }
        for (const tag of next) {
          if (ancestors.includes(tag)) {
            continue;
          }
          ancestors.push(tag);
          toCheck.push(this.allTags.filter(t => t.id === tag.parentId));
        }
      }

      // filter out tags from ancestors
      return ancestors.filter(t => !tags.includes(t.id));
    },
    withAncestors(tags: TagOut[] | TagSummary[]) {
      if (!tags) {
        return [];
      }
      const ancestors = this.getAncestors(tags.map(t => t.id)).map(t => ({ ...t, ancestors: true }));

      return [...tags.map(t => ({ ...t, ancestors: false })), ...ancestors].sort((a, b) =>
        a.name.localeCompare(b.name)
      );
    },
  },
});
