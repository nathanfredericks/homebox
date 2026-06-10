import type { PermSection } from "~/composables/use-permissions";

// Route prefixes mapped to the section whose View permission the page
// requires. Per the invisibility rule, a page the user cannot see does not
// exist: navigation silently redirects home — there is no "access denied"
// screen anywhere.
const ROUTE_SECTIONS: [prefix: string, section: PermSection][] = [
  ["/items", "items"],
  ["/item/", "items"],
  ["/assets/", "items"],
  ["/a/", "items"],
  ["/label/", "items"],
  ["/scanner-ar", "items"],
  ["/reports/label-generator", "items"],
  ["/locations", "locations"],
  ["/location/", "locations"],
  ["/tags", "tags"],
  ["/tag/", "tags"],
  ["/templates", "templates"],
  ["/template/", "templates"],
  ["/maintenance", "maintenance"],
  ["/admin/users", "users"],
  ["/admin/groups", "roles"],
  ["/collection/access", "roles"],
  ["/collection/notifiers", "notifiers"],
  ["/collection/settings", "collection_settings"],
  ["/collection/entity-types", "entity_types"],
  ["/collection/tools", "tools"],
];

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

  // Hidden pages do not exist: redirect home without explanation.
  const { canAny, canAdminArea } = usePermissions();
  const match = ROUTE_SECTIONS.find(([prefix]) => to.path === prefix || to.path.startsWith(`${prefix}`));
  if (match && !canAny(match[1], "view")) {
    return navigateTo("/home");
  }
  // Edit pages additionally require the Edit action on their section.
  if (/^\/(item|location)\/[^/]+\/edit$/.test(to.path)) {
    const section = to.path.startsWith("/item/") ? "items" : "locations";
    if (!canAny(section, "edit")) {
      return navigateTo("/home");
    }
  }
  if (to.path === "/admin" && !canAdminArea.value) {
    return navigateTo("/home");
  }
});
