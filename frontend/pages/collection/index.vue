<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "@/components/Base/Container.vue";
  import { Card } from "@/components/ui/card";
  import { Button, ButtonGroup } from "@/components/ui/button";
  import { toast } from "@/components/ui/sonner";

  import MdiShieldAccount from "~icons/mdi/shield-account";
  import MdiBell from "~icons/mdi/bell";
  import MdiCog from "~icons/mdi/cog";
  import MdiShape from "~icons/mdi/shape";
  import MdiWrench from "~icons/mdi/wrench";
  import MdiDelete from "~icons/mdi/delete";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: `HomeBox | ${t("menu.collection")}` });

  const route = useRoute();
  const api = useUserApi();
  const confirm = useConfirm();
  const { can } = usePermissions();

  const currentPath = computed(() => route.path);

  // Tabs the user cannot view do not exist for them.
  const tabs = computed(() =>
    [
      {
        id: "access",
        label: "collection.tabs.access",
        to: "/collection/access",
        icon: MdiShieldAccount,
        visible: can("roles", "view"),
      },
      {
        id: "notifiers",
        label: "collection.tabs.notifiers",
        to: "/collection/notifiers",
        icon: MdiBell,
        visible: can("notifiers", "view"),
      },
      {
        id: "settings",
        label: "collection.tabs.settings",
        to: "/collection/settings",
        icon: MdiCog,
        visible: can("collection_settings", "view"),
      },
      {
        id: "entity-types",
        label: "collection.tabs.entity_types",
        to: "/collection/entity-types",
        icon: MdiShape,
        visible: can("entity_types", "view"),
      },
      {
        id: "tools",
        label: "collection.tabs.tools",
        to: "/collection/tools",
        icon: MdiWrench,
        visible: can("tools", "view"),
      },
    ].filter(tab => tab.visible)
  );

  const { selectedCollection, load: reloadCollections } = useCollections();

  const actionLoading = ref(false);

  const canDelete = computed(() => can("collection_settings", "delete"));

  const handleDeleteCollection = async () => {
    if (!selectedCollection.value) return;

    const result = await confirm.open(t("collection.delete_confirm"));
    if (result.isCanceled) {
      return;
    }

    actionLoading.value = true;

    try {
      const res = await api.group.delete(selectedCollection.value.id);
      if (res.error) {
        const msg = t("errors.api_failure") + String(res.error);
        toast.error(msg);
        return;
      }

      toast.success(t("collection.deleted_collection"));
      await reloadCollections();
      window.location.reload();
    } catch {
      toast.error(t("errors.api_failure"));
    } finally {
      actionLoading.value = false;
    }
  };
</script>

<template>
  <BaseContainer class="print:my-0 print:max-w-none print:px-0">
    <Title>{{ t("menu.collection_options") }}</Title>

    <section class="print:hidden">
      <Card class="p-3">
        <header>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 class="text-2xl">
                {{
                  selectedCollection?.name
                    ? t("collection.manage_collection") + " - " + selectedCollection.name
                    : t("global.loading")
                }}
              </h1>
            </div>
          </div>
        </header>
      </Card>

      <div class="my-3 flex flex-wrap items-center justify-between gap-2">
        <ButtonGroup>
          <Button
            v-for="tab in tabs"
            :key="tab.id"
            as-child
            :variant="currentPath.startsWith(tab.to) ? 'default' : 'outline'"
            size="sm"
          >
            <NuxtLink :to="tab.to" class="flex items-center gap-2">
              <component :is="tab.icon" v-if="tab.icon" class="size-4" />
              <span class="hidden sm:block">{{ t(tab.label) }}</span>
            </NuxtLink>
          </Button>
        </ButtonGroup>

        <div id="collection-header-actions" class="ml-auto flex items-center gap-1">
          <Button
            v-if="canDelete"
            variant="outline"
            size="icon"
            class="size-8"
            :aria-label="$t('collection.delete_collection')"
            :disabled="!selectedCollection || actionLoading"
            @click="handleDeleteCollection"
          >
            <MdiDelete class="size-4" />
          </Button>
        </div>
      </div>
    </section>

    <section>
      <div class="space-y-6">
        <NuxtPage />
      </div>
    </section>
  </BaseContainer>
</template>
