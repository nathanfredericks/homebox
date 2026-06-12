<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
  import { Button } from "@/components/ui/button";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import { toast } from "@/components/ui/sonner";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { useDialog } from "~/components/ui/dialog-provider";
  import MdiDelete from "~icons/mdi/delete";
  import MdiPencil from "~icons/mdi/pencil";
  import MdiPlus from "~icons/mdi/plus";
  import AdminGroupCreateModal from "~/components/Admin/GroupCreateModal.vue";
  import type { RoleOut } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: t("admin.tabs.groups") });

  const api = useUserApi();
  const confirm = useConfirm();
  const { can } = usePermissions();
  const { openDialog } = useDialog();

  const deleting = ref<Record<string, boolean>>({});

  // Fetched during SSR so the table renders without a loading state.
  const { data: groupsData, refresh: refreshGroups } = await useAsyncData("admin-groups", async () => {
    const res = await api.roles.getAll();
    return res.error ? [] : (res.data ?? []);
  });
  const groups = computed<RoleOut[]>(() => groupsData.value ?? []);

  const handleCreate = () => {
    openDialog(DialogID.AdminGroupCreate, {
      onClose: id => {
        if (id) void navigateTo(`/admin/groups/${id}`);
      },
    });
  };

  const handleDelete = async (group: RoleOut) => {
    const result = await confirm.open(t("admin.groups.delete_confirm", { name: group.name }));
    if (result.isCanceled) return;

    deleting.value = { ...deleting.value, [group.id]: true };
    try {
      const res = await api.roles.delete(group.id);
      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
      } else {
        toast.success(t("admin.groups.deleted"));
        await refreshGroups();
      }
    } finally {
      deleting.value = { ...deleting.value, [group.id]: false };
    }
  };
</script>

<template>
  <div class="space-y-4">
    <div class="flex justify-end">
      <Button v-if="can('roles', 'create')" size="sm" @click="handleCreate">
        <MdiPlus class="mr-1 size-4" />
        {{ $t("admin.groups.create_group") }}
      </Button>
    </div>

    <div class="scroll-bg overflow-x-auto rounded-md border bg-card">
      <Table class="min-w-[560px]">
        <TableHeader>
          <TableRow>
            <TableHead>{{ $t("admin.groups.name") }}</TableHead>
            <TableHead>{{ $t("admin.groups.description") }}</TableHead>
            <TableHead class="w-24 text-center">{{ $t("admin.groups.members") }}</TableHead>
            <TableHead class="w-32 text-right"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="group in groups" :key="group.id">
            <TableCell>{{ group.name }}</TableCell>
            <TableCell class="text-muted-foreground">{{ group.description }}</TableCell>
            <TableCell class="text-center">{{ group.userCount }}</TableCell>
            <TableCell>
              <div class="ml-auto flex justify-end gap-1">
                <TooltipProvider :delay-duration="0">
                  <Tooltip v-if="can('roles', 'edit') && !group.isSuperAdmin">
                    <TooltipTrigger as-child>
                      <Button
                        variant="outline"
                        size="icon"
                        :aria-label="$t('global.edit')"
                        @click="navigateTo(`/admin/groups/${group.id}`)"
                      >
                        <MdiPencil class="size-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{{ $t("global.edit") }}</TooltipContent>
                  </Tooltip>
                  <Tooltip v-if="can('roles', 'delete') && !group.isSuperAdmin">
                    <TooltipTrigger as-child>
                      <Button
                        variant="destructive"
                        size="icon"
                        :aria-label="$t('global.delete')"
                        :disabled="deleting[group.id]"
                        @click="handleDelete(group)"
                      >
                        <MdiDelete class="size-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{{ $t("global.delete") }}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <AdminGroupCreateModal />
  </div>
</template>
