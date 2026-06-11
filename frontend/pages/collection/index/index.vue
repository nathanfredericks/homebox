<script setup lang="ts">
  definePageMeta({
    middleware: ["auth"],
  });

  // Land on the first section the user can see; users with no visible
  // sections never get a Collection nav entry, so fall back home.
  const { sections } = useCollectionSections();
  await navigateTo(sections.value[0]?.to ?? "/home", { replace: true });
</script>

<template>
  <div></div>
</template>
