FROM node:22-bookworm-slim AS build

WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN corepack pnpm@11.19.0 install --frozen-lockfile --ignore-scripts
COPY tsconfig.json ./
COPY src ./src
RUN corepack pnpm@11.19.0 run build && corepack pnpm@11.19.0 prune --prod --ignore-scripts

FROM node:22-bookworm-slim

WORKDIR /app
COPY --from=build /app/package.json ./
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist

ENTRYPOINT ["node", "dist/index.js"]
