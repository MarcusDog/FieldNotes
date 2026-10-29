using System;
using System.Collections;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

namespace FieldNotes
{
    public static class IssueApi
    {
        // The same key and frozen payload are reused after a timeout. The server
        // commits its deduplication record in the same file as the new issue.
        public static IEnumerator Create(string endpoint, string json, string key, Action<Issue, string> complete)
        {
            using (var request = new UnityWebRequest(endpoint + "/api/issues", "POST"))
            {
                request.uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json));
                request.downloadHandler = new DownloadHandlerBuffer();
                request.SetRequestHeader("Content-Type", "application/json");
                request.SetRequestHeader("Idempotency-Key", key);
                request.timeout = 15;
                yield return request.SendWebRequest();
                if (request.result != UnityWebRequest.Result.Success)
                {
                    complete(null, ReadError(request));
                    yield break;
                }
                Issue issue = null;
                try { issue = JsonUtility.FromJson<IssueResponse>(request.downloadHandler.text)?.data; }
                catch (ArgumentException) { /* A malformed response must not clear the draft. */ }
                if (issue == null || string.IsNullOrWhiteSpace(issue.id))
                    complete(null, "服务响应无效，保留原记录，可安全重试。请确认连接的是 FieldNotes 后端。");
                else complete(issue, null);
            }
        }

        public static IEnumerator Health(string endpoint, Action<string> complete)
        {
            using (var request = UnityWebRequest.Get(endpoint + "/api/health"))
            {
                request.timeout = 8;
                yield return request.SendWebRequest();
                if (request.result != UnityWebRequest.Result.Success) { complete(ReadError(request)); yield break; }
                HealthResponse response = null;
                try { response = JsonUtility.FromJson<HealthResponse>(request.downloadHandler.text); }
                catch (ArgumentException) { }
                complete(response?.status == "ok" ? null : "响应不是 FieldNotes 健康检查，请检查地址。");
            }
        }

        static string ReadError(UnityWebRequest request)
        {
            if (request.responseCode > 0)
            {
                try
                {
                    var error = JsonUtility.FromJson<ErrorResponse>(request.downloadHandler.text)?.error;
                    if (!string.IsNullOrEmpty(error?.message)) return $"提交失败（HTTP {request.responseCode}）：{error.message}";
                }
                catch (ArgumentException) { }
                return $"请求失败（HTTP {request.responseCode}），请检查服务器。原记录已保留。";
            }
            return $"连接失败或超时（{request.error}）。请检查同一 Wi-Fi、电脑 IP、防火墙与本地网络权限。原记录已保留，可安全重试。";
        }
    }
}
