<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "@/components/Base/Container.vue";
  import { Card } from "@/components/ui/card";

  import MdiShieldCrown from "~icons/mdi/shield-crown";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: `HomeBox | ${t("admin.title")}` });

  const route = useRoute();
  const { sections } = useAdminSections();

  // The section the current route lives in; its name is the page title.
  const currentSection = computed(() =>
    sections.value.find(section => route.path === section.to || route.path.startsWith(`${section.to}/`))
  );

  const heading = computed(() => (currentSection.value ? t(currentSection.value.labelKey) : t("admin.title")));
</script>

<template>
  <BaseContainer>
    <section>
      <Card class="p-3">
        <header>
          <div class="flex flex-wrap items-end gap-2">
            <div
              class="mb-auto flex size-12 items-center justify-center rounded-full bg-secondary text-secondary-foreground"
            >
              <component :is="currentSection?.icon ?? MdiShieldCrown" class="size-7" />
            </div>
            <div>
              <h1 class="text-wrap pb-1 text-2xl">
                {{ heading }}
              </h1>
              <div class="text-xs text-muted-foreground">
                {{ t("admin.title") }}
              </div>
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
