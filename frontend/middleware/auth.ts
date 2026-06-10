export default defineNuxtRouteMiddleware(async to => {
  const ctx = useAuthContext();
  const redirectTo = useState("authRedirect");

  if (!ctx.isAuthorized()) {
    if (to.path !== "/") {
      console.debug("[middleware/auth] isAuthorized returned false, redirecting to /");
      redirectTo.value = to.fullPath;
      return navigateTo("/");
    }
    return;
  }

  if (!ctx.user) {
    console.log("Fetching user data");
    const api = useUserApi();
    const { data, error } = await api.user.self();
    if (error) {
      if (to.path !== "/") {
        console.debug("[middleware/user] user is null and fetch failed, redirecting to /");
        redirectTo.value = to.fullPath;
        return navigateTo("/");
      }
      return;
    }

    ctx.user = data.item;
  }
});
