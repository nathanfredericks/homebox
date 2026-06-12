<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
  import { Button } from "@/components/ui/button";
  import { Badge } from "@/components/ui/badge";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import { toast } from "@/components/ui/sonner";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { useDialog } from "~/components/ui/dialog-provider";
  import MdiDelete from "~icons/mdi/delete";
  import MdiPencil from "~icons/mdi/pencil";
  import MdiPlus from "~icons/mdi/plus";
  import AdminUserCreateModal from "~/components/Admin/UserCreateModal.vue";
  import AdminUserEditModal from "~/components/Admin/UserEditModal.vue";
  import type { UserAdminOut } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: t("admin.tabs.users") });

  const api = useUserApi();
  const auth = useAuthContext();
  const confirm = useConfirm();
  const { can } = usePermissions();
  const { openDialog } = useDialog();

  const deleting = ref<Record<string, boolean>>({});

  const currentUserId = computed(() => auth.user?.id ?? "");

  // Fetched during SSR so the table (and the modals' group list) render
  // without a loading state.
  const { data: usersData, refresh: refreshUsers } = await useAsyncData("admin-users", async () => {
    const res = await api.adminUsers.getAll();
    return res.error ? [] : (res.data ?? []);
  });
  const users = computed(() => usersData.value ?? []);

  const { data: rolesData } = await useAsyncData("admin-user-roles", async () => {
    const res = await api.roles.getAll();
    return res.error ? [] : (res.data ?? []);
  });
  const roles = computed(() => rolesData.value ?? []);

  const handleCreate = () => {
    openDialog(DialogID.AdminUserCreate, {
      onClose: result => {
        if (result) void refreshUsers();
      },
    });
  };

  const handleEdit = (user: UserAdminOut) => {
    openDialog(DialogID.AdminUserEdit, {
      params: { user },
      onClose: result => {
        if (result) void refreshUsers();
      },
    });
  };

  const handleDelete = async (user: UserAdminOut) => {
    const result = await confirm.open(t("admin.users.delete_confirm", { name: user.name }));
    if (result.isCanceled) return;

    deleting.value = { ...deleting.value, [user.id]: true };
    try {
      const res = await api.adminUsers.delete(user.id);
      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
      } else {
        toast.success(t("admin.users.deleted"));
        await refreshUsers();
      }
    } finally {
      deleting.value = { ...deleting.value, [user.id]: false };
    }
  };
</script>

<template>
  <div class="space-y-4">
    <div class="flex justify-end">
      <Button v-if="can('users', 'create')" size="sm" @click="handleCreate">
        <MdiPlus class="mr-1 size-4" />
        {{ $t("admin.users.create_user") }}
      </Button>
    </div>

    <div>
      <div v-if="!users.length" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
        {{ $t("admin.users.empty") }}
      </div>

      <div v-else class="scroll-bg overflow-x-auto rounded-md border bg-card">
        <Table class="min-w-[560px]">
          <TableHeader>
            <TableRow>
              <TableHead>{{ $t("admin.users.name") }}</TableHead>
              <TableHead>{{ $t("admin.users.email") }}</TableHead>
              <TableHead>{{ $t("admin.users.groups") }}</TableHead>
              <TableHead class="w-32 text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="user in users" :key="user.id">
              <TableCell>
                {{ user.name }}
                <span v-if="user.id === currentUserId" class="ml-1 text-xs text-muted-foreground">
                  ({{ $t("admin.users.you") }})
                </span>
              </TableCell>
              <TableCell>{{ user.email }}</TableCell>
              <TableCell>
                <div class="flex flex-wrap gap-1">
                  <Badge v-for="role in user.roles" :key="role.id" variant="secondary">
                    {{ role.name }}
                  </Badge>
                </div>
              </TableCell>
              <TableCell>
                <div class="ml-auto flex justify-end gap-1">
                  <TooltipProvider :delay-duration="0">
                    <Tooltip v-if="can('users', 'edit')">
                      <TooltipTrigger as-child>
                        <Button variant="outline" size="icon" :aria-label="$t('global.edit')" @click="handleEdit(user)">
                          <MdiPencil class="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{{ $t("global.edit") }}</TooltipContent>
                    </Tooltip>
                    <Tooltip v-if="can('users', 'delete') && user.id !== currentUserId">
                      <TooltipTrigger as-child>
                        <Button
                          variant="destructive"
                          size="icon"
                          :aria-label="$t('global.delete')"
                          :disabled="deleting[user.id]"
                          @click="handleDelete(user)"
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
    </div>

    <AdminUserCreateModal :roles="roles" />
    <AdminUserEditModal :roles="roles" />
  </div>
</template>
