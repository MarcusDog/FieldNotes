# FieldNotes · 现场巡检问题标注

用 iPhone 在现场水平平面放置 AR 标记、填写问题，Go 保存记录，React 管理端查看并更新状态。手机与网页使用同一个后端。项目不需要大模型或 API Key。

## 版本

| 部分 | 使用版本 |
| --- | --- |
| Unity | 6000.3.27f1 LTS，包含 iOS Build Support |
| AR Foundation / ARKit | 6.0.8 / 6.0.8 |
| Go | 1.26，本机验证为 1.26.1，无第三方 Go 依赖 |
| React / TypeScript / Vite | 19.2.0 / 5.9.3 / 7.3.7 |
| Node.js | 22.12+，建议 24 LTS；本机使用 25.8.1 |
| 真机 | iPhone 15 Pro Max，Xcode 27.1；部署最低 iOS 15 |

## 启动 Go 与 React

安装 Go 和 Node.js 后，在仓库根目录运行：

```bash
./scripts/dev.sh
```

打开 [管理端](http://localhost:5173)。Go 默认监听 8090，数据保存到 `data/issues.json`，首次启动为空列表。Ctrl+C 同时停止两个服务；`PORT=8091 ./scripts/dev.sh` 可换端口。

也可以分别启动。下面两个终端都从仓库根目录开始：

```bash
# 终端一
cd server
ADDR=:8090 DATA_FILE=../data/issues.json go run ./cmd/fieldnotes
```

```bash
# 终端二
cd web
npm ci
API_PROXY_TARGET=http://127.0.0.1:8090 npm run dev
```

Go 单独运行的默认端口和 Vite 的默认代理都是 8080。一键脚本统一改为 8090，避免本机端口冲突。手机与网页必须指向同一进程、同一份数据。生产构建和单端口托管方法见 [Go 说明](server/README.md) 与 [React 说明](web/README.md)。

## 启动 Unity 与 iPhone

1. 用 Unity Hub 打开 `unity/`，选择上述 LTS，等待包导入。在 `File > Build Profiles` 切换到 iOS。
2. 执行 `FieldNotes > 1. Configure project and scene`。如输入系统要求重启，重启后再执行一次。在 `XR Plug-in Management > iOS` 确认 ARKit 已启用，打开 `Assets/FieldNotes/Scenes/Inspection.unity`。
3. 执行 `FieldNotes > 2. Export iOS Xcode project`，或在终端运行 `./scripts/unity.sh ios`。后者分两次启动 Editor，确保 ARKit 原生插件进入导出工程。
4. Xcode 打开 `unity/Builds/iOS/Unity-iPhone.xcodeproj`，为 `Unity-iPhone` 选择自己的签名 Team 和唯一 Bundle ID。连接、解锁并信任手机，开启开发者模式，选择真机后 Run。仓库不包含签名证书和描述文件。
5. 手机与电脑连接同一 Wi-Fi。运行 Go，在手机“后端连接设置”填写 `http://电脑局域网IP:8090`，保存并测试连接，允许相机和本地网络访问。可在电脑系统设置中查看 Wi-Fi 地址；手机上的 `localhost` 指手机自身。
6. 缓慢扫描光线充足、有纹理的桌面或地面。**绿色轮廓线表示检测到的平面，线内可轻点放置立方体**。填写标题、可选描述和优先级，提交后在电脑刷新列表。

点击表单、按钮或键盘时会阻止 AR 放置。上报坐标是平面命中点的 Unity 世界坐标，单位为米。超时后保留原草稿，重试使用同一 UUID，后端据此去重。相机被拒绝时可点击相机权限按钮进入设置，返回后重检。

## API

写请求使用 `Content-Type: application/json`。

| 方法与路径 | 用途 | 成功响应 |
| --- | --- | --- |
| `GET /api/health` | 运行检查 | `200 {"status":"ok"}` |
| `POST /api/issues` | 创建问题 | `201 {"data":Issue}` |
| `GET /api/issues` | 列出问题，最新在前 | `200 {"data":Issue[]}` |
| `PATCH /api/issues/{id}/status` | 修改状态 | `200 {"data":Issue}` |

```json
{"title":"入口墙面破损","description":"左侧墙体存在裂缝","priority":"high","position":{"x":0.42,"y":0.03,"z":1.26}}
```

后端补充 `id`、默认 `status:"open"`、UTC `createdAt/updatedAt`。状态请求为 `{"status":"in_progress"}`。优先级支持 `low/medium/high`，状态支持 `open/in_progress/resolved`。参数错误返回 400 和 `{"error":{"code":"validation_error","message":"...","fields":{...}}}`。完整约束与其他状态码见 [接口契约](docs/SPEC.md)。

## 完成情况与验证

已实现 AR 平面放置和上报、JSON 持久化、管理端列表/筛选/详情/状态处理、错误提示、安全重试和自动化测试。真机已成功上报，React 已查到同一记录，Go 实际重启后数据保留。

```bash
./scripts/verify.sh      # Go race/vet、React 测试/构建、服务重启测试
./scripts/unity.sh test  # Unity EditMode，需要已激活的 Unity
```

Unity 网络集成测试还需设置 `FIELDNOTES_TEST_API=http://127.0.0.1:8091`，指向使用独立数据文件的 Go 服务；未设置会跳过这一项。它会创建测试记录，请勿指向需要保留的现场数据。

本地交付包包含手机与电脑两段短视频，分别约 26.5 秒和 63.3 秒，展示 AR 放置、提交及网页刷新后的同一问题；本次按本人确认使用现有素材，视频未公开上传。原题每段 3～5 分钟的时长要求未达到，状态修改和服务重启也未在当前素材中展示。完整核对见 [交付检查](docs/DELIVERY_CHECK.md)，已有测试与故障修正见 [验证记录](docs/VERIFICATION.md)。

目前仅支持 iOS ARKit；平面只有绿色轮廓，部分环境不易看清。不恢复跨会话物理位置，无登录、照片上传、离线队列或自动刷新；未提交草稿在进程退出后不保留。JSON 存储适合单进程原型，不能让多个进程同时操作同一文件。损坏的数据文件会使服务拒绝启动，维护前应先停服务并备份。

[AI_LOG.md](AI_LOG.md) 按技术流程整理提示词，并保留对应原文；[技术提示词模板](docs/PROMPT_TEMPLATES.md)供后续开发使用；[核心代码讲解](docs/EXPLAIN.md)与[提交清单](docs/SUBMISSION.md)供本人复查。Docker 配置已提供但尚未实际构建；CI 状态可在仓库 Actions 查看。中文字体的 OFL 许可证随源码保留。
