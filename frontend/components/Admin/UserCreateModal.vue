<template>
  <BaseModal :dialog-id="DialogID.AdminUserCreate" :title="$t('admin.users.create_user')" :hide-footer="true">
    <form class="flex min-w-0 flex-col gap-4" @submit.prevent="submit">
      <FormTextField v-model="form.name" :label="$t('admin.users.name')" :required="true" :max-length="255" />
      <FormTextField v-model="form.email" :label="$t('admin.users.email')" type="email" :required="true" />
      <FormPassword v-model="form.password" :label="$t('admin.users.password')" :required="true" />

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
            {{ $t("global.create") }}
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

  const { t } = useI18n();
  const { activeDialog, closeDialog } = useDialog();
  const api = useUserApi();

  const loading = ref(false);
  const roles = ref<RoleSummary[]>([]);
  const form = reactive<{ name: string; email: string; password: string; roleIds: string[] }>({
    name: "",
    email: "",
    password: "",
    roleIds: [],
  });

  watch(
    () => activeDialog.value,
    async active => {
      if (active === DialogID.AdminUserCreate) {
        form.name = "";
        form.email = "";
        form.password = "";
        form.roleIds = [];
        loading.value = false;

        const res = await api.roles.getAll();
        roles.value = res.error ? [] : (res.data ?? []);
      }
    }
  );

  async function submit() {
    if (loading.value) return;
    loading.value = true;

    try {
      const res = await api.adminUsers.create({
        name: form.name,
        email: form.email,
        password: form.password,
        roleIds: form.roleIds,
      });

      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
        return;
      }

      toast.success(t("admin.users.created"));
      closeDialog(DialogID.AdminUserCreate, true);
    } catch (e) {
      toast.error((e as Error).message ?? String(e));
    } finally {
      loading.value = false;
    }
  }
</script>
