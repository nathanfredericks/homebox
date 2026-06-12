<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import MdiAccount from "~icons/mdi/account";
  import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
  import { Button } from "@/components/ui/button";
  import AuthPageShell from "~/components/App/AuthPageShell.vue";
  import FormTextField from "~/components/Form/TextField.vue";
  import FormPassword from "~/components/Form/Password.vue";
  import FormCheckbox from "~/components/Form/Checkbox.vue";
  import PasswordScore from "~/components/global/PasswordScore.vue";
  import { PASSWORD_MIN_LENGTH, PASSWORD_RULES } from "~/lib/passwords";

  const { t } = useI18n();

  useHead({
    title: t("index.title"),
  });

  definePageMeta({
    layout: "empty",
    middleware: [
      () => {
        const ctx = useAuthContext();
        if (ctx.isAuthorized()) {
          return "/home";
        } else {
          console.log("Logged out, clearing collectionId preference");
          const prefs = useViewPreferences();
          if (prefs.value.collectionId) {
            prefs.value.collectionId = null;
          }
        }
      },
    ],
  });

  const ctx = useAuthContext();

  const api = usePublicApi();
  // Use ref for OIDC error state management
  const oidcError = ref<string | null>(null);
  const shownErrorMessage = ref(false);

  const status = useAppStatus();
  const branding = useBranding();

  // Side effects from the status response are browser-only and must run after
  // the onMounted below has parsed any OIDC error from the URL.
  function applyStatusEffects() {
    if (status.value?.demo) {
      username.value = "demo@example.com";
      password.value = "demodemo";
      email.value = "demo@example.com";
      loginPassword.value = "demodemo";
    }

    // Auto-redirect to OIDC if autoRedirect is enabled, but not if there's an OIDC initialization error
    if (
      status.value?.oidc?.enabled &&
      status.value?.oidc?.autoRedirect &&
      !oidcError.value &&
      !shownErrorMessage.value
    ) {
      loginWithOIDC();
    }
  }

  whenever(status, applyStatusEffects);

  const route = useRoute();
  const router = useRouter();

  const username = ref("");
  const email = ref("");
  const password = ref("");
  const canRegister = ref(false);
  const remember = ref(false);

  // First-time setup: shown instead of the login form while no user exists.
  const showSetup = computed(() => status.value?.setup === true);

  async function setupAdmin() {
    loading.value = true;

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

    if (!emailRegex.test(email.value)) {
      toast.error(t("index.toast.invalid_email"));
      loading.value = false;
      return;
    }

    const { error } = await api.register({
      name: username.value,
      email: email.value,
      password: password.value,
    });

    if (error) {
      toast.error(t("index.toast.problem_registering"), {
        classes: {
          title: "login-error",
        },
      });
      loading.value = false;
      return;
    }

    toast.success(t("setup.success"));

    // Log the new administrator straight in.
    loginPassword.value = password.value;
    await login();
  }

  onMounted(() => {
    // Handle OIDC error notifications from URL parameters
    const oidcErrorParam = route.query.oidc_error;
    if (typeof oidcErrorParam === "string" && oidcErrorParam.startsWith("oidc_")) {
      // Set the error state to prevent auto-redirect
      oidcError.value = oidcErrorParam;
      shownErrorMessage.value = true;

      const translationKey = `index.toast.${oidcErrorParam}`;
      let errorMessage = t(translationKey);

      // If there are additional details, append them
      const details = route.query.details;
      if (typeof details === "string" && details.trim() !== "") {
        errorMessage += `: ${details}`;
      }

      toast.error(errorMessage);

      // Clean up the URL by removing the error parameters
      const newQuery = { ...route.query };
      delete newQuery.oidc_error;
      delete newQuery.details;
      router.replace({ query: newQuery });

      // Clear the error state after showing the message
      oidcError.value = null;
    }

    // With SSR the status is already resolved before mount, so the watcher
    // above never fires on initial load — apply the effects here instead.
    applyStatusEffects();
  });

  const loading = ref(false);
  const loginPassword = ref("");
  const redirectTo = useState("authRedirect");

  async function login() {
    loading.value = true;
    const { error } = await ctx.login(api, email.value, loginPassword.value, remember.value);

    if (error) {
      toast.error(t("index.toast.invalid_email_password"), {
        classes: {
          title: "login-error",
        },
      });
      loading.value = false;
      return;
    }

    toast.success(t("index.toast.login_success"));

    navigateTo(redirectTo.value || "/home");
    redirectTo.value = null;
    loading.value = false;
  }

  function loginWithOIDC() {
    window.location.href = "/api/v1/users/login/oidc";
  }
</script>

<template>
  <AuthPageShell>
    <Transition name="slide-fade">
      <form v-if="showSetup" id="setup-form" name="setup" method="post" @submit.prevent="setupAdmin">
        <Card class="md:w-[500px]">
          <CardHeader>
            <CardTitle class="flex items-center gap-2">
              <MdiAccount class="mr-1 size-7" />
              {{ $t("setup.title", { appName: branding.appName.value }) }}
            </CardTitle>
            <p class="text-sm text-muted-foreground">{{ $t("setup.subtitle") }}</p>
          </CardHeader>
          <CardContent class="flex flex-col gap-2">
            <FormTextField
              id="register-email"
              v-model="email"
              :label="$t('setup.email')"
              type="email"
              name="email"
              autocomplete="username"
              :required="true"
              data-testid="email-input"
            />
            <FormTextField
              id="register-name"
              v-model="username"
              :label="$t('setup.name')"
              name="name"
              autocomplete="name"
              :required="true"
              data-testid="name-input"
            />
            <FormPassword
              id="register-password"
              v-model="password"
              :label="$t('setup.password')"
              name="new-password"
              autocomplete="new-password"
              :min-length="PASSWORD_MIN_LENGTH"
              :passwordrules="PASSWORD_RULES"
              :required="true"
              data-testid="password-input"
            />
            <PasswordScore v-model:valid="canRegister" :password="password" />
          </CardContent>
          <CardFooter>
            <Button
              data-testid="confirm-register-button"
              class="w-full"
              type="submit"
              :class="loading ? 'loading' : ''"
              :disabled="loading || !canRegister"
            >
              {{ $t("setup.submit") }}
            </Button>
          </CardFooter>
        </Card>
      </form>
      <form v-else id="login-form" name="login" method="post" @submit.prevent="login">
        <Card class="md:w-[500px]">
          <CardHeader>
            <CardTitle class="flex items-center gap-2">
              <MdiAccount class="mr-1 size-7" />
              {{ $t("index.login") }}
            </CardTitle>
          </CardHeader>
          <CardContent v-if="status?.oidc?.allowLocal !== false" class="flex flex-col gap-2">
            <template v-if="status && status.demo">
              <p class="text-center text-xs italic">
                {{ $t("global.demo_instance") }}
              </p>
              <p class="text-center text-xs">
                <b>{{ $t("global.email") }}</b> demo@example.com
              </p>
              <p class="text-center text-xs">
                <b>{{ $t("global.password") }}</b> demodemo
              </p>
            </template>
            <FormTextField
              id="login-username"
              v-model="email"
              :label="$t('global.email')"
              name="username"
              autocomplete="username"
              :required="true"
            />
            <FormPassword
              id="login-password"
              v-model="loginPassword"
              :label="$t('global.password')"
              name="password"
              autocomplete="current-password"
              :required="true"
            />
            <div class="flex items-center justify-between">
              <div class="max-w-[140px]">
                <FormCheckbox v-model="remember" :label="$t('index.remember_me')" />
              </div>
              <NuxtLink to="/forgot-password" class="text-sm hover:underline">
                {{ $t("index.forgot_password") }}
              </NuxtLink>
            </div>
          </CardContent>
          <CardFooter class="flex flex-col gap-2">
            <Button
              v-if="status?.oidc?.allowLocal !== false"
              class="w-full"
              type="submit"
              :class="loading ? 'loading' : ''"
              :disabled="loading"
            >
              {{ $t("index.login") }}
            </Button>

            <div
              v-if="status?.oidc?.enabled && status?.oidc?.allowLocal !== false"
              class="flex w-full items-center gap-2"
            >
              <hr class="flex-1" />
              <span class="text-xs text-muted-foreground">{{ $t("index.or") }}</span>
              <hr class="flex-1" />
            </div>

            <Button v-if="status?.oidc?.enabled" type="button" variant="outline" class="w-full" @click="loginWithOIDC">
              {{ status.oidc.buttonText || "Sign in with OIDC" }}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Transition>
  </AuthPageShell>
</template>

<style lang="css" scoped>
  .slide-fade-enter-active {
    transition: all 0.2s ease-out;
  }

  .slide-fade-enter-from,
  .slide-fade-leave-to {
    position: absolute;
    transform: translateX(20px);
    opacity: 0;
  }

  progress[value]::-webkit-progress-value {
    transition: width 0.5s;
  }
</style>
