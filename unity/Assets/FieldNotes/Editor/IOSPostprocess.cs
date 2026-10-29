#if UNITY_IOS
using System.IO;
using UnityEditor;
using UnityEditor.Callbacks;
using UnityEditor.iOS.Xcode;

namespace FieldNotes.Editor
{
    public static class IOSPostprocess
    {
        [PostProcessBuild(100)]
        public static void OnBuild(BuildTarget target, string path)
        {
            if (target != BuildTarget.iOS) return;
            var plistPath = Path.Combine(path, "Info.plist");
            var plist = new PlistDocument();
            plist.ReadFromFile(plistPath);
            var root = plist.root;
            root.SetString("NSCameraUsageDescription", "使用相机识别巡检现场平面并放置问题标记。");
            root.SetString("NSLocalNetworkUsageDescription", "连接同一 Wi-Fi 下的 FieldNotes Go 服务，提交巡检问题。");
            // Local development exception only; do not disable ATS globally.
            var ats = root.values.ContainsKey("NSAppTransportSecurity") ? root["NSAppTransportSecurity"].AsDict() : root.CreateDict("NSAppTransportSecurity");
            ats.SetBoolean("NSAllowsArbitraryLoads", false);
            ats.SetBoolean("NSAllowsLocalNetworking", true);
            // iOS 17+ supports IP/CIDR exceptions. Limit development HTTP to
            // private LAN ranges; public endpoints must use normal HTTPS.
            var domains = ats.values.ContainsKey("NSExceptionDomains") ? ats["NSExceptionDomains"].AsDict() : ats.CreateDict("NSExceptionDomains");
            foreach (var range in new[] { "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16" })
            {
                var entry = domains.values.ContainsKey(range) ? domains[range].AsDict() : domains.CreateDict(range);
                entry.SetBoolean("NSExceptionAllowsInsecureHTTPLoads", true);
            }
            var capabilities = root.values.ContainsKey("UIRequiredDeviceCapabilities") ? root["UIRequiredDeviceCapabilities"].AsArray() : root.CreateArray("UIRequiredDeviceCapabilities");
            var hasARKit = false;
            foreach (var item in capabilities.values) if (item.AsString() == "arkit") hasARKit = true;
            if (!hasARKit) capabilities.AddString("arkit");
            plist.WriteToFile(plistPath);
        }
    }
}
#endif
