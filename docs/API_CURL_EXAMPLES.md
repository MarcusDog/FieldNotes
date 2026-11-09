# FieldNotes API 联调：隔离数据与可复制的 curl 请求

本示例对应 `server/internal/inspection/http.go` 和 [接口契约](SPEC.md)。目标是帮助调试手机、Go 和 React 是否访问同一个后端，并验证“提交超时后安全重试”。**仅在本机临时数据上运行**，这些 POST/PATCH 会真实写入数据。无需访问生产服务器或使用现场记录。

## 1. 启动一个与日常数据隔离的本机服务

在仓库根目录打开终端 A：

```bash
TMP_DIR="$(mktemp -d)"
(cd server && ADDR=127.0.0.1:8091 DATA_FILE="$TMP_DIR/issues.json" go run ./cmd/fieldnotes)
```

仅监听本机 `127.0.0.1`，不会对局域网开放；`TMP_DIR` 内的文件只供本次联调使用。结束时按 Ctrl+C，检查临时目录路径后手动删除该测试目录。手机真机联调则需要另行配置 LAN 地址，不能使用此本机监听方式。**切勿把 `DATA_FILE` 指向实际巡检数据。**

## 2. 检查健康与初始列表

在终端 B：

```bash
curl -i http://127.0.0.1:8091/api/health
curl -i http://127.0.0.1:8091/api/issues
```

预期分别为 HTTP 200、`{"status":"ok"}`，以及 HTTP 200、`{"data":[]}`。如果返回连接失败，先检查终端 A 是否启动成功及 8091 端口是否被占用。

## 3. 创建问题并验证幂等重试

```bash
curl -i -X POST http://127.0.0.1:8091/api/issues \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000' \
  -d '{"title":"[联调] 入口墙面破损","description":"只用于本机测试","priority":"high","position":{"x":0.42,"y":0.03,"z":1.26}}'
```

预期 HTTP 201，响应 `data` 包含新生成的 `id`、`status: "open"` 和 UTC 时间。**原样重复执行**该命令，仍应返回 HTTP 201 和相同的 `id`，列表仅新增一条。此 UUID 只用于演示：每个真实新问题都应生成新的 UUID，超时重试才复用旧 UUID。

使用同一个 `Idempotency-Key`，但将 `title` 改为另一值再提交，预期 HTTP 409 `idempotency_conflict`，不会新建第二条。这验证“同键同内容重试”和“同键不同内容拒绝”的区别。

## 4. 更新状态、查看持久化结果

复制步骤 3 响应中的 `id`，替换下面的占位符：

```bash
ISSUE_ID='paste-id-from-create-response'
curl -i -X PATCH "http://127.0.0.1:8091/api/issues/$ISSUE_ID/status" \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}'
curl -i http://127.0.0.1:8091/api/issues
```

预期 PATCH 返回 HTTP 200，列表中的对应记录显示 `in_progress`。合法状态只有 `open`、`in_progress`、`resolved`；未知 ID 返回 404，非法状态返回 400。停止再启动**同一 `TMP_DIR` 数据文件**后，记录仍应存在。

## 5. 排查浏览器 CORS 预检

```bash
curl -i -X OPTIONS http://127.0.0.1:8091/api/issues \
  -H 'Origin: http://localhost:5173' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: Content-Type, Idempotency-Key'
```

默认配置下预期 HTTP 204、`Access-Control-Allow-Origin: http://localhost:5173`，且允许 `Content-Type, Idempotency-Key`。若前端 origin 的协议、主机或端口不同，需显式设置 `CORS_ORIGINS`；禁止为了排查问题而在公共网络开放无认证服务。

## 6. 无副作用的自动化替代

若只需要验证 API、错误码、跨进程重启和持久化，而不想手工创建记录，仓库根目录直接运行：

```bash
./scripts/smoke.sh
```

该脚本自行创建临时目录、启动本地 Go 服务、验证幂等冲突、状态更新和重启，再清理测试数据。运行需要 Go 1.26+ 和 Python 3；它**不会**执行 Unity 真机 AR 测试。
