# Stage 1: сборка Nuxt
FROM node:26-alpine AS builder

WORKDIR /build

COPY package.json package-lock.json ./
RUN npm ci

COPY . ./

ENV NUXT_PUBLIC_API_BASE=/api
RUN npm run build

# Stage 2: nginx
FROM nginx:1.27-alpine

# Удаляем ВСЁ дефолтное содержимое
RUN rm -rf /usr/share/nginx/html/* /etc/nginx/conf.d/default.conf /etc/nginx/http.d/default.conf 2>/dev/null; true

COPY --from=builder /build/.output/public /usr/share/nginx/html

COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 3000

CMD ["nginx", "-g", "daemon off;"]
