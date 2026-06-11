<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
  import { Badge } from "@/components/ui/badge";
  import { Button } from "@/components/ui/button";
  import MdiPencil from "~icons/mdi/pencil";
  import type { RoleOut } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: `HomeBox | ${t("collection.tabs.access")}` });

  const api = useUserApi();
  const { can } = usePermissions();
  const { selectedId } = useCollections();

  // Fetched during SSR; failures render the inline error block below.
  const { data: rolesData, status } = await useAsyncData("collection-access", async () => {
    const res = await api.roles.getAll();
    if (res.error) {
      return { failed: true, groups: [] as RoleOut[] };
    }
    return { failed: false, groups: res.data ?? [] };
  });

  const loading = computed(() => status.value === "pending");
  const loadFailed = computed(() => rolesData.value?.failed === true);
  const groups = computed<RoleOut[]>(() => rolesData.value?.groups ?? []);

  // Which groups grant anything on the current collection, and what.
  type AccessRow = { group: RoleOut; sections: string[]; full: boolean };

  const rows = computed<AccessRow[]>(() => {
    const collectionId = selectedId.value;
    const out: AccessRow[] = [];

    for (const group of groups.value) {
      if (group.isSuperAdmin) {
        out.push({ group, sections: [], full: true });
        continue;
      }

      const sections = new Set<string>();
      for (const p of group.permissions ?? []) {
        const applies = p.collectionId == null || p.collectionId === collectionId;
        const grantsAnything = p.canView || p.canCreate || p.canEdit || p.canDelete;
        if (applies && grantsAnything) {
          sections.add(p.section);
        }
      }
      if (sections.size > 0) {
        out.push({ group, sections: [...sections], full: false });
      }
    }

    return out;
  });
</script>

<template>
  <div class="space-y-4">
    <p class="text-sm text-muted-foreground">{{ $t("collection.access.description") }}</p>

    <div v-if="loading" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
      {{ $t("global.loading") }}
    </div>

    <div v-else>
      <div v-if="loadFailed" class="rounded-md border bg-card p-4 text-sm text-destructive">
        {{ $t("errors.load_failed") }}
      </div>

      <div v-else-if="!rows.length" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
        {{ $t("collection.access.empty") }}
      </div>

      <div v-else class="scroll-bg overflow-x-auto rounded-md border bg-card">
        <Table class="min-w-[480px]">
          <TableHeader>
            <TableRow>
              <TableHead>{{ $t("collection.access.group") }}</TableHead>
              <TableHead>{{ $t("collection.access.sections") }}</TableHead>
              <TableHead class="w-24 text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in rows" :key="row.group.id">
              <TableCell>{{ row.group.name }}</TableCell>
              <TableCell>
                <Badge v-if="row.full" variant="default">{{ $t("collection.access.all_sections") }}</Badge>
                <div v-else class="flex flex-wrap gap-1">
                  <Badge v-for="section in row.sections" :key="section" variant="secondary">
                    {{ $t(`permissions.sections.${section}`) }}
                  </Badge>
                </div>
              </TableCell>
              <TableCell>
                <div class="flex justify-end">
                  <Button
                    v-if="can('roles', 'edit') && !row.group.isSuperAdmin"
                    variant="outline"
                    size="icon"
                    :aria-label="$t('global.edit')"
                    @click="navigateTo(`/admin/groups/${row.group.id}`)"
                  >
                    <MdiPencil class="size-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  </div>
</template>
