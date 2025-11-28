# ===== 构建阶段：用 Node 打包 Vue/Vite =====
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

# ===== 运行阶段：用 Nginx 提供静态页面 =====
FROM nginx:alpine

# 删除默认配置
RUN rm /etc/nginx/conf.d/default.conf

# 简单的前端配置（静态页面由 Nginx 提供，/api 走反向代理）
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 复制打包好的静态文件
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]