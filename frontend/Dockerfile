# Node dependencies stage
FROM public.ecr.aws/docker/library/node:22-alpine AS dependencies
WORKDIR /app

# Install pnpm 10 (latest stable, works reliably in Alpine)
RUN npm install -g pnpm@10

# Copy package.json and lockfile to leverage caching
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Build Nuxt (SSR) stage
FROM public.ecr.aws/docker/library/node:22-alpine AS builder
WORKDIR /app

# Install pnpm 10 (latest stable)
RUN npm install -g pnpm@10

# Copy over source files and node_modules from dependencies stage
COPY . .
COPY --from=dependencies /app/node_modules ./node_modules
RUN pnpm build

# Production stage: run the Nitro server-side rendering server
FROM public.ecr.aws/docker/library/node:22-alpine
ENV NODE_ENV=production
# Where the Nitro server reaches the Homebox API during server-side renders.
# Point this at the Go backend container.
ENV NUXT_API_HOST=http://homebox-api:7745
ENV NITRO_PORT=3000
ENV NITRO_HOST=0.0.0.0

RUN apk --no-cache add wget

WORKDIR /app
COPY --from=builder /app/.output ./.output

# Labels and configuration for the final image
LABEL Name=homebox-frontend
LABEL org.opencontainers.image.source="https://github.com/sysadminsmedia/homebox"

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD [ "wget", "--no-verbose", "--tries=1", "-O", "-", "http://localhost:3000/" ]

CMD [ "node", ".output/server/index.mjs" ]
