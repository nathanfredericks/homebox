<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Label } from "@/components/ui/label";
  import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
  import MdiLoading from "~icons/mdi/loading";
  import FormTextField from "~/components/Form/TextField.vue";
  import type { CurrenciesCurrency, Group } from "~~/lib/api/types/data-contracts";
  import { fmtCurrencyAsync } from "~/composables/utils";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: t("collection.tabs.general") });

  const api = useUserApi();
  const { selectedCollection, selectedId, load: reloadCollections } = useCollections();

  const saving = ref(false);

  const name = ref("");
  const currencyCode = ref("USD");
  const currencyExample = ref("$1,000.00");

  type SettingsData = {
    failed: boolean;
    group: Group | null;
    currencies: CurrenciesCurrency[];
  };

  // Fetched during SSR; re-runs client-side only when the selected collection
  // changes. Failures render the inline error block — no toasts on load.
  const {
    data: settings,
    status,
    refresh,
  } = await useAsyncData<SettingsData | null>(
    "collection-settings",
    async () => {
      if (!selectedId.value) {
        return null;
      }

      const [currenciesRes, groupRes] = await Promise.all([api.group.currencies(), api.group.get(selectedId.value)]);

      if (currenciesRes.error || groupRes.error || !groupRes.data) {
        return { failed: true, group: null, currencies: [] };
      }

      return { failed: false, group: groupRes.data, currencies: currenciesRes.data ?? [] };
    },
    { watch: [selectedId] }
  );

  const loading = computed(() => status.value === "pending");
  const loadFailed = computed(() => settings.value?.failed === true);
  const group = computed(() => settings.value?.group ?? null);
  const currencies = computed(() => settings.value?.currencies ?? []);

  watch(
    group,
    g => {
      if (g) {
        name.value = g.name;
        currencyCode.value = g.currency;
      }
    },
    { immediate: true }
  );

  watch(
    currencyCode,
    async () => {
      if (!currencyCode.value) return;
      try {
        currencyExample.value = await fmtCurrencyAsync(1000, currencyCode.value, getLocaleCode());
      } catch {
        currencyExample.value = `${currencyCode.value} 1000`;
      }
    },
    { immediate: true }
  );

  const save = async () => {
    if (!selectedCollection.value) return;

    saving.value = true;

    try {
      const res = await api.group.update(
        {
          name: name.value,
          currency: currencyCode.value,
        },
        selectedCollection.value.id
      );

      if (res.error || !res.data) {
        toast.error(t("profile.toast.failed_update_group"));
        return;
      }

      setCurrency(res.data.currency);
      toast.success(t("profile.toast.group_updated"));

      await reloadCollections();
      await refresh();
    } catch {
      toast.error(t("profile.toast.failed_update_group"));
    } finally {
      saving.value = false;
    }
  };
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
      {{ $t("global.loading") }}
    </div>

    <div v-else>
      <div v-if="!selectedCollection" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
        {{ $t("components.collection.selector.select_collection") }}
      </div>

      <div v-else-if="loadFailed" class="rounded-md border bg-card p-4 text-sm text-destructive">
        {{ $t("errors.load_failed") }}
      </div>

      <div v-else class="space-y-4 rounded-md border bg-card p-4">
        <FormTextField v-model="name" :label="$t('global.name')" />

        <div>
          <Label for="currency"> {{ $t("profile.currency_format") }} </Label>
          <Select
            id="currency"
            :model-value="currencyCode"
            @update:model-value="val => (currencyCode = String(val || ''))"
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="c in currencies" :key="c.code" :value="c.code">
                {{ c.name }}
              </SelectItem>
            </SelectContent>
          </Select>
          <p class="m-2 text-sm">{{ $t("profile.example") }}: {{ currencyExample }}</p>
        </div>

        <div class="mt-4">
          <Button variant="secondary" size="sm" :disabled="saving" @click="save">
            <MdiLoading v-if="saving" class="mr-2 inline-block animate-spin" />
            <span>{{ $t("global.save") }}</span>
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
