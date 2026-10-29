using System.Collections;
using System.Runtime.InteropServices;
using UnityEngine;

namespace FieldNotes
{
    public enum CameraAccess { NotDetermined = 0, Granted = 1, Denied = 2, Restricted = 3 }

    public static class CameraPermissions
    {
#if UNITY_IOS && !UNITY_EDITOR
        [DllImport("__Internal")] static extern int FN_CameraAuthorizationStatus();
        [DllImport("__Internal")] static extern void FN_RequestCameraAuthorization();
#endif
        public static CameraAccess State
        {
            get
            {
#if UNITY_IOS && !UNITY_EDITOR
                return (CameraAccess)FN_CameraAuthorizationStatus();
#else
                return CameraAccess.Granted;
#endif
            }
        }

        public static IEnumerator Request()
        {
#if UNITY_IOS && !UNITY_EDITOR
            if (State == CameraAccess.NotDetermined)
            {
                FN_RequestCameraAuthorization();
                while (State == CameraAccess.NotDetermined) yield return null;
            }
#endif
            yield break;
        }

        public static void OpenSettings()
        {
#if UNITY_IOS && !UNITY_EDITOR
            Application.OpenURL("app-settings:");
#endif
        }
    }
}
