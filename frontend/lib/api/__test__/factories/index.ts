import { faker } from "@faker-js/faker";
import { expect } from "vitest";
import { overrideParts } from "../../base/urls";
import { PublicApi } from "../../public";
import type {
  EntityFieldData,
  EntityTemplateCreate,
  TagCreate,
  EntityCreate,
  RolePermissionInput,
  UserRegistration,
} from "../../types/data-contracts";
import * as config from "../../../../test/config";
import { UserClient } from "../../user";
import { Requests } from "../../../requests";

function itemField(id = null): EntityFieldData {
  return {
    // @ts-expect-error - not actually an issue
    id,
    name: faker.lorem.word(),
    type: "text",
    textValue: faker.lorem.sentence(),
    booleanValue: false,
    numberValue: faker.number.int(),
  };
}

/**
 * Returns a random user registration object that can be
 * used to signup a new user.
 */
function user(): UserRegistration {
  return {
    email: faker.internet.email(),
    password: faker.internet.password(),
    name: faker.person.firstName(),
  };
}

function location(parentId: string | null = null): EntityCreate {
  // entityTypeId is omitted so the server resolves the default type; an empty
  // string would fail UUID decoding.
  return {
    parentId,
    name: faker.location.city(),
    description: faker.lorem.sentence(),
    manufacturer: "",
    modelNumber: "",
    notes: "",
    quantity: 1,
    serialNumber: "",
    tagIds: [],
  } as unknown as EntityCreate;
}

function item(parentId: string): EntityCreate {
  return {
    parentId,
    name: faker.commerce.productName(),
    description: faker.lorem.sentence(),
    manufacturer: faker.company.name(),
    modelNumber: faker.string.alphanumeric(10),
    notes: "",
    quantity: 1,
    serialNumber: faker.string.alphanumeric(12),
    tagIds: [],
  } as unknown as EntityCreate;
}

function tag(): TagCreate {
  return {
    name: faker.lorem.word(),
    description: faker.lorem.sentence(),
    color: faker.color.rgb(),
    icon: "",
  };
}

function template(): EntityTemplateCreate {
  return {
    name: faker.lorem.words(2),
    description: faker.lorem.sentence(),
    notes: "",
    defaultQuantity: 1,
    defaultInsured: false,
    defaultName: faker.lorem.word(),
    defaultDescription: faker.lorem.sentence(),
    defaultManufacturer: faker.company.name(),
    defaultModelNumber: faker.string.alphanumeric(10),
    defaultLifetimeWarranty: false,
    defaultWarrantyDetails: "",
    defaultLocationId: null,
    defaultTagIds: null,
    includeWarrantyFields: false,
    includePurchaseFields: false,
    includeSoldFields: false,
    fields: [],
  };
}

function publicClient(): PublicApi {
  overrideParts(config.BASE_URL, "/api/v1");
  const requests = new Requests("");
  return new PublicApi(requests);
}

function userClient(token: string): UserClient {
  overrideParts(config.BASE_URL, "/api/v1");
  const requests = new Requests("", token);
  return new UserClient(requests, "");
}

type TestUser = {
  client: UserClient;
  user: UserRegistration;
};

// Self-registration only exists for first-time setup, so all test users are
// provisioned by a bootstrapped administrator account with fixed credentials
// (login-or-setup keeps this stable across test files and workers).
const ADMIN = {
  name: "Test Admin",
  email: "test-admin@example.com",
  password: "test-admin-password",
};

let adminToken = "";

async function bootstrapAdmin(): Promise<UserClient> {
  if (adminToken) {
    return userClient(adminToken);
  }

  const pub = publicClient();
  let login = await pub.login(ADMIN.email, ADMIN.password);
  if (login.status !== 200) {
    // Empty database: run first-time setup, which makes ADMIN the Super Admin.
    await pub.register({ name: ADMIN.name, email: ADMIN.email, password: ADMIN.password });
    login = await pub.login(ADMIN.email, ADMIN.password);
  }

  expect(login.error).toBeFalsy();
  expect(login.status).toBe(200);

  adminToken = login.data.token;
  return userClient(adminToken);
}

const COLLECTION_SECTIONS = [
  "items",
  "locations",
  "tags",
  "templates",
  "maintenance",
  "statistics",
  "collection_settings",
  "entity_types",
  "notifiers",
  "tools",
];
const SITE_SECTIONS = ["users", "roles", "collections"];

function fullPermissions(): RolePermissionInput[] {
  return [...COLLECTION_SECTIONS, ...SITE_SECTIONS].map(section => ({
    section,
    collectionId: null,
    canView: true,
    canCreate: true,
    canEdit: true,
    canDelete: true,
  }));
}

async function userSingleUse(): Promise<TestUser> {
  const admin = await bootstrapAdmin();

  // Fresh collection per test user keeps tests isolated; the client pins it
  // via a default X-Tenant header.
  const collection = await admin.group.create("test-" + faker.string.alphanumeric(8));
  expect(collection.error).toBeFalsy();

  const role = await admin.roles.create({
    name: "everything-" + faker.string.alphanumeric(8),
    description: "full-access test role",
    permissions: fullPermissions(),
  });
  expect(role.error).toBeFalsy();

  const usr = user();
  const created = await admin.adminUsers.create({
    name: usr.name,
    email: usr.email,
    password: usr.password,
    roleIds: [role.data.id],
  });
  expect(created.error).toBeFalsy();

  const pub = publicClient();
  const result = await pub.login(usr.email, usr.password);

  expect(result.error).toBeFalsy();
  expect(result.status).toBe(200);

  const requests = new Requests("", result.data.token, { "X-Tenant": collection.data.id! });
  return {
    client: new UserClient(requests, result.data.attachmentToken),
    user: usr,
  };
}

export const factories = {
  user,
  location,
  item,
  tag,
  template,
  itemField,
  client: {
    public: publicClient,
    user: userClient,
    admin: bootstrapAdmin,
    singleUse: userSingleUse,
  },
};
