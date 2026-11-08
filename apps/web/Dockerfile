FROM node:26.13.0-alpine AS build
WORKDIR /src
RUN npm install -g pnpm@12.18.0
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json ./apps/web/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile --filter @cnh/web
COPY tsconfig.base.json ./
COPY apps/web ./apps/web
ARG VITE_DASH0_ENDPOINT
ARG VITE_DASH0_DATASET
RUN --mount=type=secret,id=VITE_DASH0_AUTH_TOKEN,env=VITE_DASH0_AUTH_TOKEN pnpm --filter @cnh/web build

FROM caddy:2.11-alpine
COPY apps/web/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/apps/web/dist /srv
ENV XDG_CONFIG_HOME=/tmp/config XDG_DATA_HOME=/tmp/data
USER 65532:65532
EXPOSE 8080
