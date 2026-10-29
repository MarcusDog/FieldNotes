using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.XR.ARFoundation;
using UnityEngine.XR.ARSubsystems;
using UnityEngine.XR.Management;

namespace FieldNotes
{
    public class InspectionController : MonoBehaviour
    {
        ARRaycastManager raycasts;
        ARPlaneManager planes;
        RuntimeUI ui;
        GameObject sessionRoot, originRoot, marker;
        Vector3 point;
        Material markerMaterial;
        readonly List<ARRaycastHit> hits = new List<ARRaycastHit>();
        readonly List<RaycastResult> uiHits = new List<RaycastResult>();
        string endpoint, frozenJson, requestKey, frozenEndpoint;
        bool busy, ready, abandonArmed, initializingAR;
        const string AddressPreference = "FieldNotes.Backend";

        public void Configure(GameObject session, GameObject origin, ARRaycastManager raycastManager, ARPlaneManager planeManager)
        {
            sessionRoot = session; originRoot = origin; raycasts = raycastManager; planes = planeManager;
        }

        void Start()
        {
            ui = gameObject.AddComponent<RuntimeUI>();
            endpoint = PlayerPrefs.GetString(AddressPreference, "");
            ui.Build(Submit, NewDraft, SaveAddress, CheckHealth, RetryCamera, endpoint);
            StartCoroutine(InitializeAR());
        }

        IEnumerator InitializeAR()
        {
            if (initializingAR || ready) yield break;
            initializingAR = true;
            ui.SetTracking("等待相机授权 · 请在系统弹窗中选择允许。");
            yield return CameraPermissions.Request();
            Debug.Log("FieldNotes camera authorization: " + CameraPermissions.State);
            if (CameraPermissions.State != CameraAccess.Granted)
            {
                ShowCameraPermissionHint();
                initializingAR = false;
                yield break;
            }
            if (XRGeneralSettings.Instance == null || XRGeneralSettings.Instance.Manager == null || XRGeneralSettings.Instance.Manager.activeLoader == null)
            {
                ui.SetTracking("AR 模块加载失败 · 请更新应用并检查 iOS ARKit 构建配置。");
                initializingAR = false;
                yield break;
            }
            yield return ARSession.CheckAvailability();
            if (ARSession.state == ARSessionState.Unsupported)
            {
                ui.SetTracking("此环境没有可用 ARKit。请使用支持 AR 的 iPhone 真机运行。");
                initializingAR = false;
                yield break;
            }
            sessionRoot.SetActive(true);
            originRoot.SetActive(true);
            ready = true;
            initializingAR = false;
        }

        void ShowCameraPermissionHint()
        {
            ui.SetTracking(CameraPermissions.State == CameraAccess.Restricted
                ? "相机受系统限制 · 请检查屏幕使用时间或设备管理限制。"
                : "相机未授权 · 点击“相机权限 / 重试”前往设置开启。");
        }

        void RetryCamera()
        {
            if (initializingAR) return;
            if (CameraPermissions.State == CameraAccess.Denied || CameraPermissions.State == CameraAccess.Restricted)
            {
                ShowCameraPermissionHint();
                CameraPermissions.OpenSettings();
            }
            else if (!ready) StartCoroutine(InitializeAR());
            else ui.Message("相机已授权。请缓慢移动手机扫描平面。");
        }

        void OnApplicationFocus(bool focused)
        {
            if (!focused || ui == null || initializingAR) return;
            if (CameraPermissions.State == CameraAccess.Granted)
            {
                if (!ready) StartCoroutine(InitializeAR());
            }
            else
            {
                ready = false;
                sessionRoot.SetActive(false);
                originRoot.SetActive(false);
                ShowCameraPermissionHint();
            }
        }

        void Update()
        {
            if (ui == null || !ready) return;
            var tracking = ARSession.state == ARSessionState.SessionTracking;
            ui.SetTracking(tracking ? $"AR 正在跟踪 · 已识别 {planes.trackables.count} 个平面" : $"正在恢复跟踪 · {ARSession.notTrackingReason} · 缓慢移动手机");
            if (Input.touchCount != 1 || ui.SettingsOpen || TouchScreenKeyboard.visible) return;
            var touch = Input.GetTouch(0);
            if (!PlacementPolicy.CanPlace(OverUI(touch.position), busy, frozenJson != null, tracking, touch.phase == TouchPhase.Began)) return;
            if (!raycasts.Raycast(touch.position, hits, TrackableType.PlaneWithinPolygon))
            {
                ui.Message("没有命中已识别平面，请对准绿色边界内再轻点。", true);
                return;
            }
            foreach (var hit in hits)
            {
                var plane = planes.GetPlane(hit.trackableId);
                if (plane == null || plane.trackingState != TrackingState.Tracking || plane.alignment != PlaneAlignment.HorizontalUp) continue;
                point = hit.pose.position;
                if (marker == null)
                {
                    marker = GameObject.CreatePrimitive(PrimitiveType.Cube);
                    marker.name = "Pending issue";
                    marker.transform.localScale = new Vector3(.075f, .11f, .075f);
                    Destroy(marker.GetComponent<Collider>());
                    var source = Resources.Load<Material>("Marker");
                    markerMaterial = source != null ? new Material(source) : new Material(Shader.Find("Sprites/Default"));
                    marker.GetComponent<Renderer>().sharedMaterial = markerMaterial;
                    marker.AddComponent<MarkerLifetime>().ownedMaterial = markerMaterial;
                }
                marker.transform.SetPositionAndRotation(point + Vector3.up * .055f, hit.pose.rotation);
                UpdateMarkerColor();
                ui.SetPosition(point);
                ui.SetFormState(true, false, false);
                ui.Message("标记已放置。可轻点另一平面调整位置，再填写问题并提交。");
                return;
            }
            ui.Message("请选择已稳定跟踪的朝上水平平面。", true);
        }

        void LateUpdate() { if (marker != null && frozenJson == null) UpdateMarkerColor(); }
        void UpdateMarkerColor()
        {
            if (markerMaterial != null) markerMaterial.color = ui.Priority == "high" ? new Color(.95f, .30f, .16f) : ui.Priority == "low" ? new Color(.18f, .65f, .48f) : new Color(1f, .70f, .20f);
        }
        bool OverUI(Vector2 position)
        {
            // Raycast explicitly at the touch position. IsPointerOverGameObject
            // alone can consult last frame's pointer cache in Update.
            if (EventSystem.current == null) return true;
            uiHits.Clear();
            EventSystem.current.RaycastAll(new PointerEventData(EventSystem.current) { position = position }, uiHits);
            return uiHits.Count > 0;
        }
        void SaveAddress(string input)
        {
            if (!Endpoint.TryNormalize(input, out var normalized))
            {
                ui.Message("地址无效。格式：http://电脑局域网IP:8090，不带 /api、账号或查询参数。", true);
                return;
            }
            endpoint = normalized;
            ui.Address.text = normalized;
            PlayerPrefs.SetString(AddressPreference, endpoint);
            PlayerPrefs.Save();
        }
        void CheckHealth(string input)
        {
            if (busy) return;
            if (!Endpoint.TryNormalize(input, out var normalized)) { ui.Message("地址格式无效。请输入电脑 IP:端口，例如 192.168.1.4:8090。", true); return; }
            SaveAddress(normalized);
            StartCoroutine(Health(normalized));
        }
        IEnumerator Health(string address)
        {
            busy = true; ui.SetFormState(marker != null, true, frozenJson != null);
            ui.Message("正在测试连接…");
            yield return IssueApi.Health(address, error => ui.Message(error ?? "后端连接成功，可以开始巡检。", error != null));
            busy = false; ui.SetFormState(marker != null, false, frozenJson != null);
        }
        void Submit()
        {
            if (busy || marker == null) return;
            if (frozenJson == null)
            {
                if (!Endpoint.TryNormalize(endpoint, out _)) { ui.Message("请先在连接设置中填写电脑地址。", true); return; }
                var draft = CreateIssue.From(ui.Title.text, ui.Description.text, ui.Priority, point);
                var error = draft.ValidationError();
                if (error != null) { ui.Message(error, true); return; }
                frozenJson = JsonUtility.ToJson(draft);
                frozenEndpoint = endpoint;
                requestKey = Guid.NewGuid().ToString();
            }
            abandonArmed = false;
            StartCoroutine(Send());
        }
        IEnumerator Send()
        {
            busy = true; ui.SetFormState(true, true, true);
            ui.Message("正在上报，草稿和位置已锁定…");
            yield return IssueApi.Create(frozenEndpoint, frozenJson, requestKey, (issue, error) =>
            {
                if (error != null)
                {
                    ui.Message(error + " 重试使用原地址与原数据。", true);
                    return;
                }
                marker.name = "Issue " + issue.id;
                marker = null; markerMaterial = null;
                frozenJson = requestKey = frozenEndpoint = null;
                ui.ClearDraft();
                ui.Message($"上报成功 · {issue.title}\nID {issue.id.Substring(0, Math.Min(8, issue.id.Length))} · open");
            });
            busy = false; ui.SetFormState(marker != null, false, frozenJson != null);
        }
        void NewDraft()
        {
            if (busy) return;
            if (frozenJson != null && !abandonArmed)
            {
                abandonArmed = true;
                ui.Message("上次请求可能已保存。请先在管理端核对；再次点击新建将放弃当前重试记录。", true);
                return;
            }
            if (marker != null) Destroy(marker);
            marker = null; markerMaterial = null;
            frozenJson = requestKey = frozenEndpoint = null;
            abandonArmed = false;
            ui.ClearDraft(); ui.SetFormState(false, false, false);
            ui.Message("请轻点已识别的水平平面，开始新的问题记录。");
        }
    }

    public class MarkerLifetime : MonoBehaviour
    {
        public Material ownedMaterial;
        void OnDestroy() { if (ownedMaterial != null) Destroy(ownedMaterial); }
    }
}
