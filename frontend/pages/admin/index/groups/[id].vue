<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Button } from "@/components/ui/button";
  import { toast } from "@/components/ui/sonner";
  import FormTextField from "~/components/Form/TextField.vue";
  import PermissionMatrix from "~/components/Admin/PermissionMatrix.vue";
  import MdiArrowLeft from "~icons/mdi/arrow-left";
  import type { RolePermissionInput } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: `HomeBox | ${t("admin.tabs.groups")}` });

  const api = useUserApi();
  const route = useRoute();
  const { can } = usePermissions();

  const groupId = computed(() => route.params.id as string);

  const loading = ref(true);
  const saving = ref(false);
  const isSuperAdmin = ref(false);
  const form = reactive<{ name: string; description: string; permissions: RolePermissionInput[] }>({
    name: "",
    description: "",
    permissions: [],
  });

  const collections = ref<{ id: string; name: string }[]>([]);

  onMounted(async () => {
    const [roleRes, collectionsRes] = await Promise.all([api.roles.get(groupId.value), api.group.getAll()]);

    if (roleRes.error) {
      toast.error(t("errors.api_failure") + String(roleRes.error));
      void navigateTo("/admin/groups");
      return;
    }

    // The Super Admin group is immutable; its editor does not exist.
    if (roleRes.data.isSuperAdmin || !can("roles", "edit")) {
      void navigateTo("/admin/groups");
      return;
    }

    isSuperAdmin.value = roleRes.data.isSuperAdmin;
    form.name = roleRes.data.name;
    form.description = roleRes.data.description;
    form.permissions = (roleRes.data.permissions ?? []).map(p => ({
      section: p.section,
      collectionId: p.collectionId ?? null,
      canView: p.canView,
      canCreate: p.canCreate,
      canEdit: p.canEdit,
      canDelete: p.canDelete,
    }));

    collections.value = collectionsRes.error
      ? []
      : (collectionsRes.data ?? []).map(g => ({ id: g.id!, name: g.name! }));

    loading.value = false;
  });

  async function save() {
    if (saving.value) return;
    saving.value = true;

    try {
      const res = await api.roles.update(groupId.value, {
        name: form.name,
        description: form.description,
        permissions: form.permissions,
      });

      if (res.error) {
        toast.error(t("errors.api_failure") + String(res.error));
        return;
      }

      toast.success(t("admin.groups.updated"));
      void navigateTo("/admin/groups");
    } finally {
      saving.value = false;
    }
  }
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
      {{ $t("global.loading") }}
    </div>

    <template v-else>
      <div class="flex items-center justify-between gap-2">
        <Button variant="outline" size="sm" @click="navigateTo('/admin/groups')">
          <MdiArrowLeft class="mr-1 size-4" />
          {{ $t("admin.groups.back") }}
        </Button>
        <Button size="sm" :disabled="saving" @click="save">
          {{ $t("global.save") }}
        </Button>
      </div>

      <div class="grid gap-4 rounded-md border bg-card p-4 sm:grid-cols-2">
        <FormTextField v-model="form.name" :label="$t('admin.groups.name')" :required="true" :max-length="255" />
        <FormTextField v-model="form.description" :label="$t('admin.groups.description')" :max-length="1000" />
      </div>

      <PermissionMatrix v-model="form.permissions" :collections="collections" />
    </template>
  </div>
</template>
