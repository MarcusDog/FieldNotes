# FieldNotes · 现场巡检问题标注

目标设备为 iPhone 15 Pro Max。固定栈：Unity LTS/C#/AR Foundation/ARKit、Go、React/TypeScript。基础功能优先，不接入大模型。

## 交付验收

- 真机水平平面检测，触摸命中平面才放置标记；UI 操作不能穿透放置。
- 手机填写 title（必填）、description（可选）、priority（low/medium/high），上报标记 position{x,y,z}；成功失败明确提示。
- Go 健康检查、创建、列表、按 ID 更新状态，生成唯一 ID 与时间，默认 open；明确验证参数；重启保留数据；React 可跨域访问。
- React 加载真实列表，显示标题优先级状态，更新 open/in_progress/resolved；服务不可用时有错误和重试。
- 三端共享字段 id/title/description/priority/status/position/createdAt/updatedAt。
- 完整源码、版本和启动说明、真机后端配置、完成/未完成/已知问题、按实际顺序的 AI_LOG、两段 3–5 分钟真实演示视频、可下载链接与完整 Git 历史。
- 真实 AR 与模拟请求必须清晰区分，不能伪造录屏或真机验收。项目开始时未安装 Unity、未连接手机；后续实际安装、签名与真机进展以 `VERIFICATION.md` 为准，录像与物理现场操作需要用户参与。

## 约定 API（v1）

- `GET /api/health` → `200 {"status":"ok"}`
- `POST /api/issues` body `{title,description,priority,position:{x,y,z}}` → `201 {"data":Issue}`
- `GET /api/issues` → `200 {"data":Issue[]}`（最新在前，空数组不是 null）
- `PATCH /api/issues/{id}/status` body `{"status":"in_progress"}` → `200 {"data":Issue}`
- 错误 `{ "error": { "code":"validation_error", "message":"...", "fields":{"title":"..."} } }`
- 参数错误 400、资源不存在 404、不支持 Content-Type 415、方法错误 405、请求过大 413、持久化失败 500。
- 标题 trim 后 1–120 个 Unicode 字符，描述 trim 后不超过 2000 字符；请求上限 64 KiB。
- `POST` 可带 `Idempotency-Key`（UUID），同键同请求返回原问题，异请求返回 409；用于手机超时后的安全重试。
- 坐标为本次 AR 会话中 Unity 世界坐标，单位米，不是 GPS，不保证跨会话复位。

## 实现约束

单 Go 进程使用 JSON 文件、互斥锁、临时文件 sync + rename 原子写；写失败不修改内存。默认端口 8080，数据目录不进 Git。React 开发代理 `/api`，Go 可托管构建产物，手机使用同一台电脑 LAN 地址。演示数据仅通过显式脚本创建，不在启动时自动塞入。

React 视觉方案：浅灰白工作台、墨色导航、单一松绿色强调；列表承担扫描，侧栏承担处理。轻量刷新反馈、列表行 hover、详情抽屉过渡，尊重 reduced-motion。无营销首页。

不代发邮件或 BOSS 消息，不发布到公网；交付文档提供提交模板，后续有明确发布授权再上传。
