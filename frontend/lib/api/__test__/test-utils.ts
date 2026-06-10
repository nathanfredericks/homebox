import { beforeAll } from "vitest";
import type { UserClient } from "../user";
import { factories } from "./factories";

const cache: { client: UserClient | null } = {
  client: null,
};

/*
 * Shared UserClient for tests where the creation of a user is _not_ important
 * to the test. Users are provisioned by the bootstrapped admin account since
 * self-registration only exists for first-time setup.
 */
export async function sharedUserClient(): Promise<UserClient> {
  if (cache.client) {
    return cache.client;
  }
  const { client } = await factories.client.singleUse();
  cache.client = client;
  return client;
}

beforeAll(async () => {
  await sharedUserClient();
});
