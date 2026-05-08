# Travel Planner Viewer (Go + AngularJS + Leaflet)

一个支持多出行计划管理的旅游行程查看与编辑工具。

当前版本支持：
- 计划列表与搜索排序
- 计划详情地图与图层切换
- 管理员模式下的计划增删改、停留点增删改
- Excel 导入生成计划（支持地点名自动定位）

---

## 1. 项目结构

```text
howPlan/
├─ backend/
│  ├─ cmd/server/main.go
│  ├─ internal/
│  ├─ data/
│  │  ├─ index.json
│  │  └─ plans/
│  └─ go.mod
├─ frontend/
│  ├─ index.html
│  └─ app/
├─ start.sh
└─ README.md
```

---

## 2. 启动方式（脚本）

只需要使用脚本启动，不需要手动执行 `go run`。

浏览模式：

```bash
./start.sh
```

管理员模式：

```bash
./start.sh true
```

说明：
- 启动脚本会在 `18080-19080` 之间自动选择可用随机端口。
- 启动后会自动用 Chrome 打开页面。
- 前端和后端同源部署，不需要单独启动前端服务。

---

## 3. 管理员模式切换

- 启动时使用 `./start.sh true` 可进入管理员模式。
- 也可以在 URL 上加 `?admin=true` 临时请求管理员能力（例如：`http://localhost:18xxx/?admin=true#!/plans`）。

---

## 4. 数据文件

计划数据写入位置：
- `backend/data/index.json`
- `backend/data/plans/*.json`

当前仓库默认包含一份真实行程计划：
- `cebu-bohol-siquijor-moalboal`

---

## 5. 常见问题

### 5.1 端口或权限问题

如果启动失败，请检查终端日志。脚本已自动规避常见端口冲突；若系统限制监听端口，请放开终端/安全软件权限。

### 5.2 数据写入失败

请确认 `backend/data` 目录对当前用户可写。

### 5.3 导入定位失败

Excel 导入时，若地点名无法自动解析，会在导入结果中显示 warning，可后续在地图上手动修正坐标。

---

## 6. API 概览

- `GET /api/admin/status`
- `GET /api/plans`
- `POST /api/plans`（管理员）
- `GET /api/plans/:id`
- `PUT /api/plans/:id`（管理员）
- `DELETE /api/plans/:id`（管理员）
- `POST /api/plans/:id/stops`（管理员）
- `DELETE /api/plans/:id/stops/:stopId`（管理员）
- `GET /api/plans/import/template`（管理员）
- `POST /api/plans/import`（管理员）
