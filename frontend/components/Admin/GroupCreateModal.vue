<template>
  <BaseModal :dialog-id="DialogID.AdminGroupCreate" :title="$t('admin.groups.create_group')" :hide-footer="true">
    <form class="flex min-w-0 flex-col gap-4" @submit.prevent="submit">
      <FormTextField v-model="form.name" :label="$t('admin.groups.name')" :required="true" :max-length="255" />
      <FormTextField v-model="form.description" :label="$t('admin.groups.description')" :max-length="1000" />

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
  import { Button, ButtonGroup } from "~/components/ui/button";
  import { toast } from "@/components/ui/sonner";
  import { useUserApi } from "~/composables/use-api";

  const { t } = useI18n();
  const { activeDialog, closeDialog } = useDialog();
  const api = useUserApi();

  const loading = ref(false);
  const form = reactive<{ name: string; description: string }>({ name: "", description: "" });

  watch(
    () => activeDialog.value,
    active => {
      if (active === DialogID.AdminGroupCreate) {
        form.name = "";
        form.description = "";
        loading.value = false;
      }
    }
  );

  async function submit() {
    if (loading.value) return;
    loading.value = true;

    try {
      const res = await api.roles.create({
        name: form.name,
        description: form.description,
        permissions: [],
      });

      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
        return;
      }

      toast.success(t("admin.groups.created"));
      closeDialog(DialogID.AdminGroupCreate, res.data.id);
    } catch {
      toast.error(t("errors.api_failure"));
    } finally {
      loading.value = false;
    }
  }
</script>
