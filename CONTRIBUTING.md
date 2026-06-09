# Contributing

## We Develop with GitHub

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## Branch Flow

We use the `main` branch as the development branch. All PRs should be made to the `main` branch from a feature branch. To create a pull request, you can use the following steps:

1. Fork the repository and create a new branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure that the test suite and linters pass
5. Issue your pull request

## How To Get Started

### Prerequisites

There is a devcontainer available for this project. If you are using VSCode, you can use the devcontainer to get started. If you are not using VSCode, you need to ensure that you have the following tools installed:

- [Go 1.19+](https://golang.org/doc/install)
- [Swaggo](https://github.com/swaggo/swag)
- [Node.js 16+](https://nodejs.org/en/download/)
- [pnpm](https://pnpm.io/installation)
- [Taskfile](https://taskfile.dev/#/installation) (Optional but recommended)
- For code generation, you'll need to have `python3` available on your path. In most cases, this is already installed and available.

If you're using `taskfile` you can run `task --list-all` for a list of all commands and their descriptions.

### Setup

If you're using the taskfile, you can use the `task setup` command to run the required setup commands. Otherwise, you can review the commands required in the `Taskfile.yml` file.

### API Development Notes

start command `task go:run`

1. API Server does not auto reload. You'll need to restart the server after making changes.
2. Unit tests should be written in Go, however, end-to-end or user story tests should be written in TypeScript using the client library in the frontend directory.

### Frontend Development Notes

start command `task ui:dev`

1. The frontend is a Next.js 15 (App Router) app written in React with Material UI for styling. It runs as a standalone Node server rather than being embedded in the Go binary.
2. In development, `task ui:dev` runs the Next.js dev server on `http://localhost:3000`. HTTP `/api/*` requests are proxied to the Go API on `http://localhost:7745` (see `frontend/next.config.ts`); the WebSocket events connection dials the Go server directly. Run the backend in another terminal with `task go:run`.
3. `task ui:build` produces the production standalone build; `task ui:check` runs the TypeScript type check and `task ui:fix` runs ESLint + Prettier.
4. The frontend is SSR-first: Next.js performs server-side data fetches against the Go API. The API base URL comes from `API_BASE_URL`, which defaults to `http://localhost:7745` in dev (SSR fetches go through the dev proxy).
5. In production (the official Docker image), Caddy serves everything on port `7745`: it routes `/api/*` (including the events WebSocket) and `/swagger*` to the Go API and everything else to the Next.js server. The Go API and Next.js server both listen on loopback only. There, `API_BASE_URL=http://127.0.0.1:7746` is set so the Next server's SSR fetches reach the Go API directly and skip the Caddy hop.

> The frontend automated test suite (Vitest/Playwright) is parked during the Next.js migration and will be restored on the new stack.

## Publishing Release

Create a new tag in GitHub with the version number vX.X.X. This will trigger a new release to be created.

Test -> Goreleaser -> Publish Release -> Trigger Docker Builds -> Deploy Docs + Fly.io Demo
