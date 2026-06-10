import type { UserClient } from "~~/lib/api/user";

export function itemsTable(api: UserClient) {
  const asyncData = useAsyncData(
    "items",
    async () => {
      const { data } = await api.items.getAll({
        page: 1,
        pageSize: 5,
        orderBy: "createdAt",
      });
      return data.items;
    },
    {
      deep: true,
    }
  );

  const { data: items, refresh } = asyncData;

  onServerEvent(ServerEvent.EntityMutation, () => {
    console.log("entity mutation");
    refresh();
  });

  const table = computed(() => {
    return {
      items: items.value || [],
    };
  });

  // asyncData is awaited by the page so the items are in the SSR payload
  return { table, asyncData };
}
