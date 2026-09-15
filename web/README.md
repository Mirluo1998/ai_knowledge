# Knowledge Web（知识管理系统前端）

Knowledge Service 的独立前端工程，与 Go 后端完全解耦：独立的 `package.json`、独立构建产物 `dist/`、独立 Dockerfile，不依赖 Go 工具链。

## 技术栈

- React 18 + TypeScript（strict 模式）
- Vite 5（构建与开发服务器）
- Ant Design 5（中文语言包 `zhCN`，dayjs 使用中文 locale）
- react-router-dom 6（路由与路由守卫）
- axios（HTTP 请求，拦截器兼容后端两种响应结构）

## 目录结构

```
web/
├── src/
│   ├── api/            # axios 实例（client.ts）与各接口封装（auth.ts、knowledge.ts）
│   ├── components/     # 通用组件：路由守卫 ProtectedRoute、全局布局 AppLayout
│   ├── pages/          # 页面：LoginPage、RegisterPage、KnowledgePage
│   ├── router/         # 路由表
│   ├── styles/         # 全局样式
│   ├── types/          # 全部接口请求/响应类型定义
│   ├── utils/          # 工具：localStorage 鉴权信息存取
│   └── main.tsx        # 入口（ConfigProvider 中文、dayjs 中文）
├── .env.development    # 开发环境变量
├── .env.production     # 生产环境变量示例
├── nginx.conf          # SPA 部署 + 同域反代示例
├── Dockerfile          # 多阶段构建（node:22-alpine → nginx:alpine）
├── vite.config.ts
└── package.json
```

## 本地开发

1. 先启动 Go 后端（监听 `:8080`），在仓库根目录：

   ```bash
   make run
   ```

2. 进入本目录安装依赖并启动开发服务器：

   ```bash
   cd web
   npm install
   npm run dev
   ```

3. 浏览器打开 <http://localhost:5173>。Vite 已配置代理，`/api` 与 `/healthz`
   会转发到 `http://localhost:8080`（`changeOrigin: true`），开发期无需后端开启 CORS。

## 构建

```bash
cd web
npm run build     # tsc -b 类型检查 + vite build，产物输出到 web/dist/
npm run preview   # 本地预览构建产物
npm run lint      # ESLint 检查
```

## 环境变量

| 变量名               | 说明                                   | 默认值     |
| -------------------- | -------------------------------------- | ---------- |
| `VITE_API_BASE_URL`  | 后端 API 前缀                          | `/api/v1`  |

- 开发环境见 `.env.development`；生产环境见 `.env.production`。
- 生产构建默认走相对路径 `/api/v1`，由部署层（Nginx）在同域下反向代理到后端。

## 部署方式

### 方式一：Nginx 静态部署 + 同域反向代理

```bash
npm run build          # 产出 dist/
# 将 dist/ 内容发布到 Nginx 的 /usr/share/nginx/html
# 使用本仓库的 nginx.conf（包含 SPA try_files 回退与 /api、/healthz 反代）
```

`nginx.conf` 默认把 `/api/` 与 `/healthz` 代理到 `knowledge-server:8080`，
后端主机名/端口不同时修改其中的 `proxy_pass` 即可。

### 方式二：Docker

```bash
# 在仓库根目录执行（构建上下文为 web/）
docker build -t knowledge-web ./web
docker run -d -p 80:80 --name knowledge-web knowledge-web
```

容器内 Nginx 监听 80 端口，同样通过 `knowledge-server:8080`
（容器网络中的后端服务名）访问后端，请确保前后端容器接入同一 Docker 网络。
