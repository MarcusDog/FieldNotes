using System;
using UnityEngine;

namespace FieldNotes
{
    [Serializable]
    public class IssuePosition
    {
        public float x, y, z;
        public bool IsFinite() => !(float.IsNaN(x) || float.IsInfinity(x) || float.IsNaN(y) || float.IsInfinity(y) || float.IsNaN(z) || float.IsInfinity(z));
    }

    [Serializable]
    public class CreateIssue
    {
        public string title;
        public string description;
        public string priority;
        public IssuePosition position;

        public static CreateIssue From(string title, string description, string priority, Vector3 point) => new CreateIssue
        {
            title = (title ?? "").Trim(), description = (description ?? "").Trim(), priority = priority,
            position = new IssuePosition { x = point.x, y = point.y, z = point.z }
        };

        public string ValidationError()
        {
            if (string.IsNullOrWhiteSpace(title)) return "请填写问题标题。";
            if (title.Length > 120) return "标题不能超过 120 字符。";
            if ((description ?? "").Length > 2000) return "描述不能超过 2000 字符。";
            if (priority != "low" && priority != "medium" && priority != "high") return "请选择有效的优先级。";
            if (position == null || !position.IsFinite()) return "标记坐标无效，请重新放置。";
            return null;
        }
    }

    [Serializable]
    public class Issue
    {
        public string id, title, description, priority, status, createdAt, updatedAt;
        public IssuePosition position;
    }
    [Serializable] public class IssueResponse { public Issue data; }
    [Serializable] public class ApiError { public string code, message; }
    [Serializable] public class ErrorResponse { public ApiError error; }
    [Serializable] public class HealthResponse { public string status; }

    public static class Endpoint
    {
        public static bool TryNormalize(string value, out string normalized)
        {
            normalized = null;
            if (!Uri.TryCreate((value ?? "").Trim(), UriKind.Absolute, out var uri) ||
                (uri.Scheme != "http" && uri.Scheme != "https") || string.IsNullOrEmpty(uri.Host) ||
                uri.UserInfo.Length > 0 || uri.Query.Length > 0 || uri.Fragment.Length > 0 || uri.AbsolutePath != "/") return false;
            normalized = uri.GetLeftPart(UriPartial.Authority).TrimEnd('/');
            return true;
        }
    }

    public static class PlacementPolicy
    {
        public static bool CanPlace(bool overUI, bool busy, bool draftLocked, bool tracking, bool touchBegan) =>
            touchBegan && tracking && !overUI && !busy && !draftLocked;
    }
}
