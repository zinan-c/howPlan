# Travel Planner Viewer (Go + AngularJS + Leaflet)

一个支持**多出行计划管理**的旅游行程查看与编辑工具。  
当前版本支持：
- 计划列表（多计划）
- 计划详情地图（按计划 ID 加载）
- 管理员模式下的计划增删改、停留点增删改

---

## 1. 项目目录结构

```text
howDive/
├─ backend/
│  ├─ cmd/
│  │  └─ server/
│  │     └─ main.go
│  ├─ internal/
│  │  ├─ handlers/
│  │  │  ├─ middleware.go
│  │  │  └─ plans.go
│  │  ├─ models/
│  │  │  └─ plan.go
│  │  ├─ store/
│  │  │  └─ plans_store.go
│  │  └─ utils/
│  │     └─ http.go
│  ├─ data/
│  │  ├─ index.json
│  │  └─ plans/
│  │     ├─ yunnan-trip.json
│  │     ├─ japan-sakura.json
│  │     └─ xinjiang-roadtrip.json
│  └─ go.mod
├─ frontend/
│  ├─ index.html
│  └─ app/
│     ├─ app.js
│     ├─ controllers/
│     │  ├─ mapController.js
│     │  └─ plansController.js
│     ├─ directives/
│     │  └─ leafletMap.js
│     ├─ services/
│     │  └─ tripService.js
│     ├─ styles/
│     │  └─ main.css
│     └─ views/
│        ├─ plans.html
│        └─ map.html
├─ start.sh
└─ README.md
```

---

## 2. 后端启动命令（管理员模式示例）

### 推荐方式（进入 backend 目录运行）

```bash
cd /Users/nic/prj/howDive/backend
ADMIN_MODE=true go run ./cmd/server/main.go
```

普通模式：

```bash
cd /Users/nic/prj/howDive/backend
ADMIN_MODE=false go run ./cmd/server/main.go
```

> 你提到的 `ADMIN_MODE=true go run main.go` 也可行，但需先 `cd` 到 `cmd/server` 并确保模块路径正确。  
> 项目内推荐统一使用 `go run ./cmd/server/main.go`。

### URL 参数临时覆盖（前端）

访问：

```text
http://localhost:8080/?admin=true
```

前端会请求 `/api/admin/status?admin=true`，如果后端允许会进入编辑能力。

---

## 3. 前端访问地址

Go 后端已托管前端静态文件，直接访问：

```text
http://localhost:8080/
```

不需要单独启动前端 dev server。

---

## 4. 测试步骤

### 4.1 普通模式测试

1. 启动：
   ```bash
   cd /Users/nic/prj/howDive/backend
   ADMIN_MODE=false go run ./cmd/server/main.go
   ```
2. 打开 `http://localhost:8080/`
3. 进入任一计划详情页（`/plan/:id`）
4. 验证：
   - 看不到新增计划按钮
   - 看不到计划卡片编辑/删除按钮
   - 地图页看不到停留点编辑/删除能力（或点击提示无权限）

### 4.2 管理员模式测试

1. 启动：
   ```bash
   cd /Users/nic/prj/howDive/backend
   ADMIN_MODE=true go run ./cmd/server/main.go
   ```
2. 打开 `http://localhost:8080/?admin=true`
3. 在计划列表页验证：
   - 可以新增计划
   - 可以编辑计划基本信息
   - 可以删除计划
   - 可以复制计划
4. 进入计划详情页验证：
   - 可以添加停留点（地图点选）
   - 可以编辑停留点（含拖拽改坐标）
   - 可以删除停留点

---

## 5. 示例数据说明

首次启动或当前仓库默认数据包含 **3 个示例计划**：

1. 云南之旅（`yunnan-trip`）
2. 日本赏樱（`japan-sakura`）
3. 新疆自驾（`xinjiang-roadtrip`）

每个计划默认包含约 **2-3 天**示例行程（停留点、活动、图片占位 URL）。

---

## 6. 常见问题

### 6.1 CORS 问题

- 如果你改成前后端分离（例如前端在 `5500`），后端已预置常见 localhost 源。
- 同域（`8080`）访问通常不会有 CORS 问题。

### 6.2 端口冲突（8080 被占用）

- 先查占用并结束冲突进程，或修改 `main.go` 中监听端口。
- macOS 常用排查：
  ```bash
  lsof -i :8080
  ```

### 6.3 文件权限问题（data 目录写入失败）

- 计划增删改会写入：
  - `backend/data/index.json`
  - `backend/data/plans/*.json`
- 请确保 `backend/data` 及其子目录对当前用户可写。

---

## API 概览（当前后端）

- `GET /api/admin/status`
- `GET /api/plans`
- `POST /api/plans`（管理员）
- `GET /api/plans/:id`
- `PUT /api/plans/:id`（管理员）
- `DELETE /api/plans/:id`（管理员）
- `POST /api/plans/:id/stops`（管理员）
- `DELETE /api/plans/:id/stops/:stopId`（管理员）

---

## 关键代码文件最终版本检查清单

后端：
- [main.go](/Users/nic/prj/howDive/backend/cmd/server/main.go)
- [plans.go](/Users/nic/prj/howDive/backend/internal/handlers/plans.go)
- [middleware.go](/Users/nic/prj/howDive/backend/internal/handlers/middleware.go)
- [plan.go](/Users/nic/prj/howDive/backend/internal/models/plan.go)
- [plans_store.go](/Users/nic/prj/howDive/backend/internal/store/plans_store.go)
- [http.go](/Users/nic/prj/howDive/backend/internal/utils/http.go)
- [index.json](/Users/nic/prj/howDive/backend/data/index.json)

前端：
- [index.html](/Users/nic/prj/howDive/frontend/index.html)
- [app.js](/Users/nic/prj/howDive/frontend/app/app.js)
- [tripService.js](/Users/nic/prj/howDive/frontend/app/services/tripService.js)
- [plansController.js](/Users/nic/prj/howDive/frontend/app/controllers/plansController.js)
- [mapController.js](/Users/nic/prj/howDive/frontend/app/controllers/mapController.js)
- [plans.html](/Users/nic/prj/howDive/frontend/app/views/plans.html)
- [map.html](/Users/nic/prj/howDive/frontend/app/views/map.html)
- [leafletMap.js](/Users/nic/prj/howDive/frontend/app/directives/leafletMap.js)
- [main.css](/Users/nic/prj/howDive/frontend/app/styles/main.css)
