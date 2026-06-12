<template>
  <BaseModal :dialog-id="DialogID.AdminUserEdit" :title="$t('admin.users.edit_user')" :hide-footer="true">
    <form class="flex min-w-0 flex-col gap-4" @submit.prevent="submit">
      <FormTextField v-model="form.name" :label="$t('admin.users.name')" :required="true" :max-length="255" />
      <FormTextField v-model="form.email" :label="$t('admin.users.email')" type="email" :required="true" />
      <FormPassword v-model="form.password" :label="$t('admin.users.new_password')" :required="false" />

      <div class="flex w-full flex-col gap-1.5">
        <Label>{{ $t("admin.users.groups") }}</Label>
        <div class="flex flex-col gap-1 rounded-md border p-3">
          <label v-for="role in roles" :key="role.id" class="flex cursor-pointer items-center gap-2 text-sm">
            <input v-model="form.roleIds" type="checkbox" class="size-4 accent-primary" :value="role.id" />
            {{ role.name }}
          </label>
          <p v-if="!roles.length" class="text-sm text-muted-foreground">{{ $t("admin.users.no_groups") }}</p>
        </div>
      </div>

      <div class="mt-4 flex flex-row-reverse">
        <ButtonGroup>
          <Button :disabled="loading" type="submit">
            {{ $t("global.save") }}
          </Button>
        </ButtonGroup>
      </div>
    </form>
  </BaseModal>
</template>

<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { useDialog } from "~/components/ui/dialog-provider";
  import BaseModal from "@/components/App/CreateModal.vue";
  import FormTextField from "~/components/Form/TextField.vue";
  import FormPassword from "~/components/Form/Password.vue";
  import { Button, ButtonGroup } from "~/components/ui/button";
  import { Label } from "~/components/ui/label";
  import { toast } from "@/components/ui/sonner";
  import { useUserApi } from "~/composables/use-api";
  import type { RoleSummary } from "~~/lib/api/types/data-contracts";

  // The role list is fetched (SSR) by the users page so it is already
  // available when the dialog opens.
  defineProps<{ roles: RoleSummary[] }>();

  const { t } = useI18n();
  const { activeDialog, closeDialog, registerOpenDialogCallback } = useDialog();
  const api = useUserApi();

  const loading = ref(false);
  const userId = ref<string>("");
  const form = reactive<{ name: string; email: string; password: string; roleIds: string[] }>({
    name: "",
    email: "",
    password: "",
    roleIds: [],
  });

  onMounted(() => {
    registerOpenDialogCallback(DialogID.AdminUserEdit, params => {
      userId.value = params.user.id;
      form.name = params.user.name;
      form.email = params.user.email;
      form.password = "";
      form.roleIds = (params.user.roles ?? []).map(r => r.id);
      loading.value = false;
    });
  });

  async function submit() {
    if (loading.value || !userId.value) return;
    loading.value = true;

    try {
      const res = await api.adminUsers.update(userId.value, {
        name: form.name,
        email: form.email,
        password: form.password,
        roleIds: form.roleIds,
      });

      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
        return;
      }

      toast.success(t("admin.users.updated"));
      closeDialog(DialogID.AdminUserEdit, true);
    } catch {
      toast.error(t("errors.api_failure"));
    } finally {
      loading.value = false;
    }
  }

  // Suppress unused warning for activeDialog (kept for parity with sibling modals).
  void activeDialog;
</script>
