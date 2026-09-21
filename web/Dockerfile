FROM node:22-alpine AS build

WORKDIR /src
ARG NPM_REGISTRY=https://registry.npmmirror.com
COPY package.json package-lock.json ./
RUN npm config set registry "${NPM_REGISTRY}" && npm ci
COPY . .
ARG VITE_BASE_PATH=/
ENV VITE_BASE_PATH=$VITE_BASE_PATH
RUN npm run build

FROM nginx:1.27-alpine

ARG VITE_BASE_PATH=/
ENV VITE_BASE_PATH=$VITE_BASE_PATH
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.d/40-base-path.sh /docker-entrypoint.d/40-base-path.sh
RUN chmod +x /docker-entrypoint.d/40-base-path.sh
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
