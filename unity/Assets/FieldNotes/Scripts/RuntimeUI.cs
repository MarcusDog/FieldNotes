using System;
using UnityEngine;
using UnityEngine.Events;
using UnityEngine.EventSystems;
using UnityEngine.UI;

namespace FieldNotes
{
    public class RuntimeUI : MonoBehaviour
    {
        public InputField Title { get; private set; }
        public InputField Description { get; private set; }
        public InputField Address { get; private set; }
        public bool SettingsOpen => settings != null && settings.activeSelf;
        public string Priority { get; private set; } = "medium";
        Text tracking, feedback, connectionFeedback, coordinates, submitLabel;
        Button submit, reset, connection;
        readonly Button[] priorities = new Button[3];
        GameObject settings;
        RectTransform safe;
        Rect lastSafe;
        Font font;
        readonly Color ink = new Color(.08f, .14f, .12f);
        readonly Color accent = new Color(.10f, .42f, .31f);
        readonly Color muted = new Color(.9f, .94f, .92f);

        public void Build(Action report, Action clear, Action<string> saveAddress, Action<string> health, string endpoint)
        {
            font = Resources.Load<Font>("NotoSansCJKsc-Regular");
            if (font == null) font = Resources.GetBuiltinResource<Font>("LegacyRuntime.ttf");
            if (EventSystem.current == null) new GameObject("Event System", typeof(EventSystem), typeof(StandaloneInputModule));
            var canvas = new GameObject("FieldNotes UI", typeof(Canvas), typeof(CanvasScaler), typeof(GraphicRaycaster));
            canvas.GetComponent<Canvas>().renderMode = RenderMode.ScreenSpaceOverlay;
            var scaler = canvas.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(1080, 1920);
            scaler.matchWidthOrHeight = .5f;
            safe = Rect("Safe area", canvas.transform);
            ApplySafeArea();

            var header = Panel("Header", safe, new Vector2(0, 1), Vector2.one, new Vector2(26, -284), new Vector2(-26, -12));
            var headerStack = Stack(header, 20);
            Label(headerStack, "FIELDNOTES  /  现场巡检", 40, 58, FontStyle.Bold);
            tracking = Label(headerStack, "正在初始化 AR…", 29, 75);
            Button(headerStack, "后端连接设置", () => settings.SetActive(true), 54);

            var panel = Panel("Report form", safe, Vector2.zero, new Vector2(1, 0), new Vector2(26, 14), new Vector2(-26, 790));
            var form = Stack(panel, 22);
            Label(form, "01  轻点绿色平面放置标记", 31, 44, FontStyle.Bold);
            coordinates = Label(form, "尚未放置 · 坐标单位：米", 24, 38);
            Title = Input(form, "标题（必填，最多 120 字符）", 120, 68);
            Description = Input(form, "描述（可选）", 2000, 106, true);
            Label(form, "02  问题优先级", 27, 38, FontStyle.Bold);
            var row = Row(form, 62);
            var names = new[] { "低 · low", "中 · medium", "高 · high" };
            var values = new[] { "low", "medium", "high" };
            for (var i = 0; i < 3; i++)
            {
                var index = i;
                priorities[i] = Button(row, names[i], () => SetPriority(values[index]), 62);
            }
            SetPriority("medium");
            feedback = Label(form, "请缓慢移动手机，扫描有纹理、光线充足的桌面或地面。", 26, 92);
            submit = Button(form, "03  提交问题", () => report(), 76, true);
            submitLabel = submit.GetComponentInChildren<Text>();
            reset = Button(form, "新建 / 清除当前草稿", () => clear(), 58);

            settings = new GameObject("Connection overlay", typeof(RectTransform), typeof(Image));
            settings.transform.SetParent(safe, false);
            Stretch(settings.GetComponent<RectTransform>());
            settings.GetComponent<Image>().color = new Color(.04f, .09f, .07f, .96f);
            var config = Panel("Connection", settings.transform, new Vector2(0, .25f), new Vector2(1, .75f), new Vector2(28, 0), new Vector2(-28, 0));
            var configStack = Stack(config, 26);
            Label(configStack, "连接 Go 后端", 42, 66, FontStyle.Bold);
            Label(configStack, "手机与电脑连接同一 Wi-Fi。输入电脑的局域网地址，不能使用 localhost。", 28, 100);
            Address = Input(configStack, "http://192.168.1.20:8090", 240, 75);
            Address.contentType = InputField.ContentType.Standard;
            Address.keyboardType = TouchScreenKeyboardType.URL;
            Address.text = endpoint;
            connection = Button(configStack, "保存并测试连接", () => { saveAddress(Address.text); health(Address.text); }, 78, true);
            connectionFeedback = Label(configStack, "开发环境仅用于可信局域网。", 27, 120);
            Button(configStack, "返回巡检", () => { saveAddress(Address.text); settings.SetActive(false); }, 65);
            settings.SetActive(string.IsNullOrEmpty(endpoint));
            SetFormState(false, false, false);
        }

        public void SetTracking(string message) { tracking.text = message; }
        public void Message(string message, bool error = false)
        {
            feedback.text = message;
            feedback.color = error ? new Color(.7f, .16f, .12f) : ink;
            if (connectionFeedback != null) { connectionFeedback.text = message; connectionFeedback.color = feedback.color; }
        }
        public void SetPosition(Vector3? point)
        {
            coordinates.text = point.HasValue ? $"X {point.Value.x:F2}   Y {point.Value.y:F2}   Z {point.Value.z:F2} m" : "尚未放置 · 坐标单位：米";
        }
        public void SetFormState(bool hasMarker, bool busy, bool locked)
        {
            Title.interactable = Description.interactable = !busy && !locked;
            foreach (var button in priorities) button.interactable = !busy && !locked;
            submit.interactable = hasMarker && !busy;
            reset.interactable = !busy;
            connection.interactable = !busy;
            submitLabel.text = busy ? "正在提交，请稍候…" : locked ? "安全重试同一问题" : "03  提交问题";
        }
        public void ClearDraft()
        {
            Title.text = Description.text = "";
            SetPriority("medium");
            SetPosition(null);
        }
        void SetPriority(string value)
        {
            Priority = value;
            var values = new[] { "low", "medium", "high" };
            for (var i = 0; i < priorities.Length; i++)
                if (priorities[i] != null)
                {
                    priorities[i].GetComponent<Image>().color = value == values[i] ? accent : muted;
                    priorities[i].GetComponentInChildren<Text>().color = value == values[i] ? Color.white : ink;
                }
        }
        void Update() { if (safe != null && Screen.safeArea != lastSafe) ApplySafeArea(); }
        void ApplySafeArea()
        {
            lastSafe = Screen.safeArea;
            safe.anchorMin = new Vector2(lastSafe.xMin / Screen.width, lastSafe.yMin / Screen.height);
            safe.anchorMax = new Vector2(lastSafe.xMax / Screen.width, lastSafe.yMax / Screen.height);
            safe.offsetMin = safe.offsetMax = Vector2.zero;
        }
        RectTransform Rect(string name, Transform parent)
        {
            var go = new GameObject(name, typeof(RectTransform));
            go.transform.SetParent(parent, false);
            return go.GetComponent<RectTransform>();
        }
        static void Stretch(RectTransform rect)
        {
            rect.anchorMin = Vector2.zero; rect.anchorMax = Vector2.one;
            rect.offsetMin = rect.offsetMax = Vector2.zero;
        }
        Transform Panel(string name, Transform parent, Vector2 min, Vector2 max, Vector2 offsetMin, Vector2 offsetMax)
        {
            var rect = Rect(name, parent);
            rect.anchorMin = min; rect.anchorMax = max; rect.offsetMin = offsetMin; rect.offsetMax = offsetMax;
            rect.gameObject.AddComponent<Image>().color = new Color(.97f, .98f, .97f, .97f);
            return rect;
        }
        Transform Stack(Transform parent, int padding)
        {
            var rect = Rect("Content", parent); Stretch(rect);
            var layout = rect.gameObject.AddComponent<VerticalLayoutGroup>();
            layout.padding = new RectOffset(padding, padding, padding, padding);
            layout.spacing = 10;
            layout.childForceExpandHeight = false;
            layout.childControlHeight = true;
            layout.childForceExpandWidth = true;
            return rect;
        }
        Transform Row(Transform parent, float height)
        {
            var rect = Rect("Priority", parent);
            rect.gameObject.AddComponent<LayoutElement>().preferredHeight = height;
            var layout = rect.gameObject.AddComponent<HorizontalLayoutGroup>();
            layout.spacing = 10; layout.childForceExpandWidth = true; layout.childControlWidth = true;
            return rect;
        }
        Text Label(Transform parent, string text, int size, float height, FontStyle style = FontStyle.Normal)
        {
            var rect = Rect("Label", parent);
            rect.gameObject.AddComponent<LayoutElement>().preferredHeight = height;
            var label = rect.gameObject.AddComponent<Text>();
            label.font = font; label.text = text; label.fontSize = size; label.fontStyle = style;
            label.color = ink; label.raycastTarget = false;
            label.verticalOverflow = VerticalWrapMode.Truncate;
            return label;
        }
        Button Button(Transform parent, string text, UnityAction action, float height, bool primary = false)
        {
            var rect = Rect(text, parent);
            rect.gameObject.AddComponent<LayoutElement>().preferredHeight = height;
            var image = rect.gameObject.AddComponent<Image>(); image.color = primary ? accent : muted;
            var button = rect.gameObject.AddComponent<Button>(); button.targetGraphic = image;
            button.onClick.AddListener(action);
            var label = Label(rect, text, 28, height);
            Stretch(label.rectTransform); label.alignment = TextAnchor.MiddleCenter;
            label.color = primary ? Color.white : ink;
            return button;
        }
        InputField Input(Transform parent, string placeholder, int limit, float height, bool multiline = false)
        {
            var rect = Rect("Input", parent);
            rect.gameObject.AddComponent<LayoutElement>().preferredHeight = height;
            rect.gameObject.AddComponent<Image>().color = Color.white;
            var field = rect.gameObject.AddComponent<InputField>();
            var label = Label(rect, "", 29, height);
            Stretch(label.rectTransform); label.rectTransform.offsetMin = new Vector2(14, 9); label.rectTransform.offsetMax = new Vector2(-14, -9);
            var hint = Label(rect, placeholder, 27, height); Stretch(hint.rectTransform);
            hint.rectTransform.offsetMin = new Vector2(14, 9); hint.rectTransform.offsetMax = new Vector2(-14, -9);
            hint.color = new Color(.37f, .43f, .40f);
            field.textComponent = label; field.placeholder = hint; field.characterLimit = limit;
            field.lineType = multiline ? InputField.LineType.MultiLineNewline : InputField.LineType.SingleLine;
            return field;
        }
    }
}
