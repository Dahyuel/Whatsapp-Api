# ── Build Stage ─────────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm config set registry https://registry.npmmirror.com/ && npm install

COPY . .
RUN npm run build

# ── serve stage ─────────────────────────────────────────────────
FROM nginx:alpine

# Copy built static files
COPY --from=builder /app/dist /usr/share/nginx/html

# Replace default nginx config with our custom one to handle React Router
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
