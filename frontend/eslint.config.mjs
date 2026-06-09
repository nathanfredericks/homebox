import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { FlatCompat } from "@eslint/eslintrc";
import prettier from "eslint-config-prettier";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  {
    ignores: [".next/**", "node_modules/**", "next-env.d.ts", "lib/api/__test__/**", "**/*.test.ts"],
  },
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    rules: {
      "no-console": "off",
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          ignoreRestSiblings: true,
          destructuredArrayIgnorePattern: "_",
          caughtErrors: "none",
          argsIgnorePattern: "^_",
        },
      ],
    },
  },
  {
    // lib/ is framework-agnostic code ported verbatim from the Vue app
    // (api client, generated types, otel). It predates the strict rules
    // applied to new app code; relax the rules it intentionally violates
    // rather than rewriting ported code.
    files: ["lib/**/*.ts"],
    linterOptions: {
      // Ported files carry Vue-era eslint-disable directives that are now
      // redundant; keep them verbatim rather than reporting them as unused.
      reportUnusedDisableDirectives: "off",
    },
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-unused-vars": "off",
    },
  },
  prettier,
];

export default eslintConfig;
