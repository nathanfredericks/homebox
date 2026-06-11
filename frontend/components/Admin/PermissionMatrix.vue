<template>
  <div class="space-y-6">
    <div v-for="group in sectionGroups" :key="group.id" class="space-y-2">
      <h3 class="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {{ t(`permissions.groups.${group.id}`) }}
      </h3>

      <div class="scroll-bg overflow-x-auto rounded-md border bg-card">
        <Table class="min-w-[640px]">
          <TableHeader>
            <TableRow>
              <TableHead>{{ t("permissions.section") }}</TableHead>
              <TableHead>{{ t("permissions.scope") }}</TableHead>
              <TableHead v-for="action in actions" :key="action" class="w-20 text-center">
                {{ t(`permissions.actions.${action}`) }}
              </TableHead>
              <TableHead class="w-12"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-for="section in group.sections" :key="section.id">
              <TableRow v-for="(row, idx) in rowsFor(section.id)" :key="`${section.id}-${idx}`">
                <TableCell :class="idx > 0 ? 'text-transparent' : ''">
                  {{ t(`permissions.sections.${section.id}`) }}
                </TableCell>
                <TableCell>
                  <span v-if="!section.collectionScoped" class="text-sm text-muted-foreground">
                    {{ t("permissions.site_wide") }}
                  </span>
                  <select
                    v-else
                    class="w-full max-w-52 rounded-md border bg-background px-2 py-1 text-sm"
                    :value="row.collectionId ?? ALL"
                    @change="setScope(row, ($event.target as HTMLSelectElement).value)"
                  >
                    <option :value="ALL">{{ t("permissions.all_collections") }}</option>
                    <option v-for="c in collections" :key="c.id" :value="c.id">{{ c.name }}</option>
                  </select>
                </TableCell>
                <TableCell v-for="action in actions" :key="action" class="text-center">
                  <input
                    v-if="section.actions.includes(action)"
                    type="checkbox"
                    class="size-4 accent-primary"
                    :checked="row[actionField(action)]"
                    @change="toggle(row, action, ($event.target as HTMLInputElement).checked)"
                  />
                </TableCell>
                <TableCell class="text-right">
                  <Button
                    variant="ghost"
                    size="icon"
                    class="size-7"
                    :aria-label="t('global.remove')"
                    @click="removeRow(row)"
                  >
                    <MdiClose class="size-4" />
                  </Button>
                </TableCell>
              </TableRow>

              <TableRow class="hover:bg-transparent">
                <TableCell v-if="rowsFor(section.id).length === 0">
                  {{ t(`permissions.sections.${section.id}`) }}
                </TableCell>
                <TableCell v-else></TableCell>
                <TableCell :colspan="actions.length + 2">
                  <Button variant="ghost" size="sm" class="h-7 text-xs text-muted-foreground" @click="addRow(section)">
                    <MdiPlus class="mr-1 size-3.5" />
                    {{ section.collectionScoped ? t("permissions.add_scope") : t("permissions.grant") }}
                  </Button>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
  import { Button } from "@/components/ui/button";
  import MdiClose from "~icons/mdi/close";
  import MdiPlus from "~icons/mdi/plus";
  import type { RolePermissionInput } from "~~/lib/api/types/data-contracts";
  import type { PermAction } from "~/composables/use-permissions";

  // The matrix mirrors the UI surfaces, grouped exactly as users experience
  // them; cells for actions a section doesn't support are simply not shown.
  type SectionDef = {
    id: string;
    collectionScoped: boolean;
    actions: PermAction[];
  };

  const ALL = "__all__";
  const actions: PermAction[] = ["view", "create", "edit", "delete"];

  const sectionGroups: { id: string; sections: SectionDef[] }[] = [
    {
      id: "inventory",
      sections: [
        { id: "items", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "locations", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "tags", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "templates", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "maintenance", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "statistics", collectionScoped: true, actions: ["view"] },
      ],
    },
    {
      id: "collection",
      sections: [
        { id: "collection_settings", collectionScoped: true, actions: ["view", "edit", "delete"] },
        { id: "entity_types", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "notifiers", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
        { id: "tools", collectionScoped: true, actions: ["view", "create", "edit", "delete"] },
      ],
    },
    {
      id: "administration",
      sections: [
        { id: "users", collectionScoped: false, actions: ["view", "create", "edit", "delete"] },
        { id: "roles", collectionScoped: false, actions: ["view", "create", "edit", "delete"] },
        { id: "collections", collectionScoped: false, actions: ["view", "create"] },
        { id: "site_settings", collectionScoped: false, actions: ["view", "edit"] },
      ],
    },
  ];

  const props = defineProps<{
    modelValue: RolePermissionInput[];
    collections: { id: string; name: string }[];
  }>();

  const emit = defineEmits<{
    "update:modelValue": [value: RolePermissionInput[]];
  }>();

  const { t } = useI18n();

  const rowsFor = (sectionId: string) => props.modelValue.filter(r => r.section === sectionId);

  const actionField = (action: PermAction) =>
    (({ view: "canView", create: "canCreate", edit: "canEdit", delete: "canDelete" }) as const)[action];

  function update(next: RolePermissionInput[]) {
    emit("update:modelValue", next);
  }

  function addRow(section: SectionDef) {
    update([
      ...props.modelValue,
      {
        section: section.id,
        collectionId: null,
        canView: false,
        canCreate: false,
        canEdit: false,
        canDelete: false,
      },
    ]);
  }

  function removeRow(row: RolePermissionInput) {
    update(props.modelValue.filter(r => r !== row));
  }

  function setScope(row: RolePermissionInput, value: string) {
    update(props.modelValue.map(r => (r === row ? { ...r, collectionId: value === ALL ? null : value } : r)));
  }

  function toggle(row: RolePermissionInput, action: PermAction, checked: boolean) {
    const field = actionField(action);
    update(props.modelValue.map(r => (r === row ? { ...r, [field]: checked } : r)));
  }
</script>
