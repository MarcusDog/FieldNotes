# 核心代码讲解

## 一次问题怎样流过三端

1. `unity/Assets/FieldNotes/Scripts/Bootstrap.cs` 创建 `ARSession`、`XROrigin`、跟踪相机、`ARPlaneManager` 和 `ARRaycastManager`。跟踪相机的姿态来自 ARKit 输入。
2. `InspectionController.Update` 只处理单指 `TouchPhase.Began`。先在触点执行 UI Raycast；有 UI 命中、键盘、设置浮层、未稳定跟踪或正在提交时直接退出。随后用 `PlaneWithinPolygon` 命中真实已跟踪且朝上的水平平面。
3. 命中点保存为 `point`，立方体显示在其上方；提交坐标来自平面命中点，而不是为了显示而抬高的立方体中心。颜色表示优先级。
4. `CreateIssue` 只发送用户字段。首次提交将 JSON、地址和 UUID 冻结；不确定失败时不改写这些数据。`IssueApi` 用 `UnityWebRequest` 发送 UTF-8 JSON，有连接超时和用户可见错误。
5. `server/internal/inspection/http.go` 严格解析并验证请求；缺字段、空标题、错误枚举、坐标不完整都返回 400。后端生成 ID、UTC 时间和默认 `open` 状态。
6. `store.go` 在一把互斥锁内构建下一份数据，写同目录临时文件，sync/close，rename，最后才更新内存。写失败不会出现“网页看见但重启丢失”的内存提交。
7. 重试键和原始响应与问题一起持久化。同键同数据返回原响应；同键换数据返回 409。并发重试只有一条问题，重启后仍能识别重试。
8. `web/src/api.ts` 请求同一个 API，并检查返回结构。`App.tsx` 管理列表、筛选、选中 ID、加载/保存状态；PATCH 成功才采用服务器返回值，失败保留原状态。

## 为什么使用本地 JSON

题目允许 JSON。小型单进程原型无需引入数据库驱动；临时文件替换和互斥锁足以避免单进程内的写入竞争与部分写入。代价是每次写全量文件、不支持多进程共享，数据量大时应迁移 SQLite 或数据库。

## 标记怎样保持世界位置

固定物体不能证明识别了真实平面。本项目的标记必须来自 ARKit 平面 raycast，移动手机后由真实姿态跟踪维持世界位置。绿色边界来自 `ARPlane.boundary`。模板本身隐藏在不激活的父对象下，但其 `activeSelf` 保持为 true，因为 AR Foundation 会把该状态复制给检测到的平面。

## 为什么有一个很小的 iOS 原生授权文件

真机反馈揭示：没有使用 WebCamTexture 的 AR 包会生成 `UNITY_USES_WEBCAM=0`，Unity 的 WebCam 授权实现可能成为空实现。我们没有增加第二路相机捕获，而是通过 `Assets/Plugins/iOS/CameraAuthorization.mm` 调用 Apple 的 AVCaptureDevice 授权 API，仅查询/请求权限，不读取图像。C# 的 `CameraPermissions` 等待系统授权状态改变后再启动 ARKit；画面与平面仍由 AR Foundation/ARKit 提供。拒绝后用户可以点击相机权限按钮打开设置，回到前台自动重检。

## 为什么命令行导出分成两次启动

ARKit 6.0.8 的批量构建路径依赖在加载脚本时就存在的 `UNITY_XR_ARKIT_LOADER_ENABLED` 符号；只有 XR 设置里勾选加载器而没有该符号时，原生插件可能被导出过滤器排除。配置步骤先保存加载器和符号，新的 Editor 进程再编译/导出，最后检查 `UnityARKit.m` 和 `libUnityARKit.a` 确实被带入。运行时若加载器为空，会提示构建配置错误，不把它直接当成手机不支持 AR。

## 为什么不能把 AR 坐标当成园区地图

每次 AR 会话有独立的原点；`x/y/z` 仅表示本次会话的 Unity 世界坐标、单位米。不做共享空间地图或持久化锚点时，两台设备的同一坐标不一定是同一个物理位置。题目明确允许不恢复历史真实物理位置，本项目未实现跨设备定位。

## 本人验收时应能解释

- UI 射线和 AR 射线有什么区别，为什么先检查 UI。
- 为什么请求失败不立即清掉草稿；重试键能处理什么、不能处理什么。
- 为什么必须持久化成功才响应成功，损坏 JSON 为什么要拒绝启动。
- React 保存失败如何保留状态，刷新与保存如何避免竞态。
- 手机 localhost 与电脑 localhost 为什么不同；开发代理、同源请求和 CORS 的关系。

这些说明用于阅读和验证代码，不代表候选人已经完成了人工检查。
