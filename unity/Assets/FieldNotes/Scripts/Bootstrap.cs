using Unity.XR.CoreUtils;
using UnityEngine;
using UnityEngine.InputSystem;
using UnityEngine.InputSystem.XR;
using UnityEngine.XR.ARFoundation;
using UnityEngine.XR.ARSubsystems;

namespace FieldNotes
{
    public class Bootstrap : MonoBehaviour
    {
        void Awake()
        {
            Application.targetFrameRate = 60;
            Screen.sleepTimeout = SleepTimeout.NeverSleep;
            var session = new GameObject("AR Session");
            session.SetActive(false);
            session.AddComponent<ARSession>();
            session.AddComponent<ARInputManager>();

            var originObject = new GameObject("XR Origin");
            originObject.SetActive(false);
            var origin = originObject.AddComponent<XROrigin>();
            var offset = new GameObject("Camera Offset");
            offset.transform.SetParent(originObject.transform, false);
            var cameraObject = new GameObject("AR Camera");
            cameraObject.transform.SetParent(offset.transform, false);
            cameraObject.tag = "MainCamera";
            var camera = cameraObject.AddComponent<Camera>();
            camera.clearFlags = CameraClearFlags.SolidColor;
            camera.backgroundColor = Color.black;
            camera.nearClipPlane = .05f; camera.farClipPlane = 50;
            cameraObject.AddComponent<ARCameraManager>();
            cameraObject.AddComponent<ARCameraBackground>();
            var driver = cameraObject.AddComponent<TrackedPoseDriver>();
            var position = new InputAction("Position", binding: "<XRHMD>/centerEyePosition", expectedControlType: "Vector3");
            position.AddBinding("<HandheldARInputDevice>/devicePosition");
            var rotation = new InputAction("Rotation", binding: "<XRHMD>/centerEyeRotation", expectedControlType: "Quaternion");
            rotation.AddBinding("<HandheldARInputDevice>/deviceRotation");
            driver.positionInput = new InputActionProperty(position);
            driver.rotationInput = new InputActionProperty(rotation);
            origin.Camera = camera;
            origin.CameraFloorOffsetObject = offset;
            origin.RequestedTrackingOriginMode = XROrigin.TrackingOriginMode.Device;
            origin.CameraYOffset = 0;

            var planes = originObject.AddComponent<ARPlaneManager>();
            planes.requestedDetectionMode = PlaneDetectionMode.Horizontal;
            planes.planePrefab = CreatePlaneTemplate();
            var raycasts = originObject.AddComponent<ARRaycastManager>();
            gameObject.AddComponent<InspectionController>().Configure(session, originObject, raycasts, planes);
        }

        public static GameObject CreatePlaneTemplate()
        {
            // ARTrackableManager preserves prefab.activeSelf on its clones.
            // Keep the source hidden through its parent, not activeSelf=false.
            var storage = new GameObject("Hidden runtime templates");
            storage.SetActive(false);
            var template = new GameObject("Plane template");
            template.transform.SetParent(storage.transform, false);
            template.AddComponent<ARPlane>();
            template.AddComponent<LineRenderer>();
            template.AddComponent<PlaneOutline>();
            return template;
        }
    }

}
