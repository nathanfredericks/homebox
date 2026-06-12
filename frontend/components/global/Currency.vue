<template>
  {{ formattedValue }}
</template>

<script setup lang="ts">
  import { computed } from "vue";

  type Props = {
    amount: string | number;
  };

  const props = defineProps<Props>();

  // Awaited in setup so the formatted value is server-rendered: the currency
  // code lands in a useState cache that serializes into the SSR payload, so
  // hydration reuses it without refetching.
  const fmt = await useFormatCurrency();

  const formattedValue = computed(() => {
    if (!props.amount || props.amount === "0") {
      return fmt(0);
    }

    return fmt(props.amount);
  });
</script>
