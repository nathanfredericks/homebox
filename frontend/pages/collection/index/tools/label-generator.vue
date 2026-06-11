<script setup lang="ts">
  definePageMeta({
    middleware: ["auth"],
  });

  // The label generator moved to its own top-level section; keep old
  // bookmarks working.
  await navigateTo("/collection/labels", { replace: true });
</script>

<template>
  <div></div>
</template>
