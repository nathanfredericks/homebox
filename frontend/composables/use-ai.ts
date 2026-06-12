/**
 * Whether AI features are enabled on this instance (admin settings → AI).
 * All AI UI must be gated with `v-if="aiEnabled && can(...)"` — hidden, never
 * disabled. The status is cached for the session; toggling the admin setting
 * applies on the next page load.
 */
export function useAi() {
  const api = useUserApi();

  const { data } = useAsyncData("ai-status", async () => {
    const { data, error } = await api.ai.status();
    if (error || !data) return false;
    return data.enabled;
  });

  const aiEnabled = computed(() => data.value === true);

  return { aiEnabled };
}
