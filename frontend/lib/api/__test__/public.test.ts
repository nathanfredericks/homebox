import { describe, expect, test } from "vitest";
import { factories } from "./factories";

describe("[GET] /api/v1/status", () => {
  test("server should respond", async () => {
    const api = factories.client.public();
    const { response, data } = await api.status();
    expect(response.status).toBe(200);
    expect(data.health).toBe(true);
  });
});

describe("first-time setup semantics", () => {
  test("registration is closed once setup has completed", async () => {
    // Ensure the admin (and therefore at least one user) exists.
    await factories.client.admin();

    const api = factories.client.public();
    const { response: statusResp, data: status } = await api.status();
    expect(statusResp.status).toBe(200);
    expect(status.setup).toBe(false);

    // Self-registration must be rejected after setup.
    const { response } = await api.register(factories.user());
    expect(response.status).toBe(403);
  });

  test("admin-created users can log in", async () => {
    const { client, user } = await factories.client.singleUse();

    const { response, data } = await client.user.self();
    expect(response.status).toBe(200);
    expect(data.item.email.toLowerCase()).toBe(user.email.toLowerCase());
    expect(data.item.permissions.length).toBeGreaterThan(0);
  });
});
