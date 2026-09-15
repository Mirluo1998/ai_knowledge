# ---- 构建阶段 ----
FROM node:22-alpine AS builder
WORKDIR /app

# 先复制依赖描述文件，利用 Docker 层缓存
COPY package*.json ./
RUN npm install

# 复制源码并构建（产物在 /app/dist）
COPY . .
RUN npm run build

# ---- 运行阶段 ----
FROM nginx:alpine

# SPA 站点配置（含 /api、/healthz 反向代理）
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 复制构建产物
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
