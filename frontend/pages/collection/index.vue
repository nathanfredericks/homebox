<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "@/components/Base/Container.vue";
  import { Card } from "@/components/ui/card";
  import { Button } from "@/components/ui/button";
  import { toast } from "@/components/ui/sonner";

  import MdiHomeGroup from "~icons/mdi/home-group";
  import MdiDelete from "~icons/mdi/delete";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: `HomeBox | ${t("collection.collection_settings")}` });

  const api = useUserApi();
  const confirm = useConfirm();
  const { can } = usePermissions();

  const route = useRoute();
  const { sections } = useCollectionSections();

  // The section the current route lives in; its name is the page title.
  const currentSection = computed(() =>
    sections.value.find(section => route.path === section.to || route.path.startsWith(`${section.to}/`))
  );

  const heading = computed(() =>
    currentSection.value ? t(currentSection.value.labelKey) : t("collection.collection_settings")
  );

  const { selectedCollection, load: reloadCollections } = useCollections();

  const subtitle = computed(
    () => `${selectedCollection.value?.name ?? t("global.loading")} · ${t("collection.collection_settings")}`
  );

  const actionLoading = ref(false);

  // Deleting the collection lives with its general settings; a destructive
  // button under e.g. a "Notifiers" heading reads as deleting that instead.
  const canDelete = computed(() => currentSection.value?.id === "settings" && can("collection_settings", "delete"));

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
    <section class="print:hidden">
      <Card class="p-3">
        <header>
          <div class="flex flex-wrap items-end gap-2">
            <div
              class="mb-auto flex size-12 items-center justify-center rounded-full bg-secondary text-secondary-foreground"
            >
              <component :is="currentSection?.icon ?? MdiHomeGroup" class="size-7" />
            </div>
            <div>
              <h1 class="text-wrap pb-1 text-2xl">
                {{ heading }}
              </h1>
              <div class="text-xs text-muted-foreground">
                {{ subtitle }}
              </div>
            </div>
            <div class="ml-auto mt-2 flex flex-wrap items-center justify-between gap-3">
              <Button
                v-if="canDelete"
                variant="destructive"
                :disabled="!selectedCollection || actionLoading"
                @click="handleDeleteCollection"
              >
                <MdiDelete />
                {{ $t("global.delete") }}
              </Button>
            </div>
          </div>
        </header>
      </Card>
    </section>

    <section class="mt-3">
      <div class="space-y-6">
        <NuxtPage />
      </div>
    </section>
  </BaseContainer>
</template>
