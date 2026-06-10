import { faker } from "@faker-js/faker";
import { describe, expect, test } from "vitest";
import { factories } from "../factories";

describe("collections (site-owned groups)", () => {
  test("user with collection_settings:edit should be able to update the collection", async () => {
    const { client } = await factories.client.singleUse();

    const name = faker.person.firstName();

    const { response, data: group } = await client.group.update({
      name,
      currency: "eur",
    });

    expect(response.status).toBe(200);
    expect(group.name).toBe(name);
  });

  test("user should be able to get the current collection", async () => {
    const { client } = await factories.client.singleUse();

    const { response, data: group } = await client.group.get();

    expect(response.status).toBe(200);
    expect(group.name).toBeTruthy();
    expect(group.currency).toBe("USD");
  });

  test("user with collections:create should be able to create a collection", async () => {
    const { client } = await factories.client.singleUse();

    const name = "created-" + faker.string.alphanumeric(8);
    const { response, data: group } = await client.group.create(name);

    expect(response.status).toBe(201);
    expect(group.name).toBe(name);

    // The new collection is visible in the accessible list (the test role
    // holds all-collections grants).
    const { data: all } = await client.group.getAll();
    expect(all.some(g => g.id === group.id)).toBe(true);
  });

  test("users and roles are managed at the site level", async () => {
    const admin = await factories.client.admin();

    const { response: usersResp, data: users } = await admin.adminUsers.getAll();
    expect(usersResp.status).toBe(200);
    expect(users.length).toBeGreaterThan(0);

    const { response: rolesResp, data: roles } = await admin.roles.getAll();
    expect(rolesResp.status).toBe(200);
    expect(roles.some(r => r.isSuperAdmin)).toBe(true);
  });
});
