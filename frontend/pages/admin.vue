<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "@/components/Base/Container.vue";
  import { Card } from "@/components/ui/card";
  import { Button, ButtonGroup } from "@/components/ui/button";

  import MdiAccountMultiple from "~icons/mdi/account-multiple";
  import MdiShieldAccount from "~icons/mdi/shield-account";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();
  const { can } = usePermissions();

  useHead({ title: `HomeBox | ${t("admin.title")}` });

  const route = useRoute();
  const currentPath = computed(() => route.path);

  // Tabs the user cannot view do not exist for them.
  const tabs = computed(() =>
    [
      {
        id: "users",
        label: "admin.tabs.users",
        to: "/admin/users",
        icon: MdiAccountMultiple,
        visible: can("users", "view"),
      },
      {
        id: "groups",
        label: "admin.tabs.groups",
        to: "/admin/groups",
        icon: MdiShieldAccount,
        visible: can("roles", "view"),
      },
    ].filter(tab => tab.visible)
  );
</script>

<template>
  <BaseContainer>
    <Title>{{ t("admin.title") }}</Title>

    <section>
      <Card class="p-3">
        <header>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h1 class="text-2xl">{{ t("admin.title") }}</h1>
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
      </div>
    </section>

    <section>
      <div class="space-y-6">
        <NuxtPage />
      </div>
    </section>
  </BaseContainer>
</template>
