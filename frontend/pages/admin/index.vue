<script setup lang="ts">
  definePageMeta({
    middleware: ["auth"],
  });

  const { can } = usePermissions();

  // Land on the first surface the user can see; the auth middleware already
  // redirected users with no admin access away.
  const target = can("users", "view") ? "/admin/users" : "/admin/groups";
  await navigateTo(target, { replace: true });
</script>

<template>
  <div></div>
</template>
