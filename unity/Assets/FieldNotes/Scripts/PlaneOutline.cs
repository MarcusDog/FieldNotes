using UnityEngine;
using UnityEngine.XR.ARFoundation;
using UnityEngine.XR.ARSubsystems;

namespace FieldNotes
{
    // Outline actual ARKit plane boundaries, rather than an invented screen plane.
    [RequireComponent(typeof(ARPlane), typeof(LineRenderer))]
    public class PlaneOutline : MonoBehaviour
    {
        ARPlane plane;
        LineRenderer line;
        Material material;
        void Awake()
        {
            plane = GetComponent<ARPlane>();
            line = GetComponent<LineRenderer>();
            line.useWorldSpace = false;
            line.loop = true;
            line.widthMultiplier = .008f;
            var source = Resources.Load<Material>("Plane");
            material = source != null ? new Material(source) : new Material(Shader.Find("Sprites/Default"));
            line.sharedMaterial = material;
            line.startColor = line.endColor = new Color(.3f, .95f, .72f, .8f);
        }
        void OnEnable() { plane.boundaryChanged += Changed; Redraw(); }
        void OnDisable() { plane.boundaryChanged -= Changed; }
        void Changed(ARPlaneBoundaryChangedEventArgs _) { Redraw(); }
        void Update()
        {
            line.enabled = plane.trackingState == TrackingState.Tracking && plane.subsumedBy == null;
        }
        void Redraw()
        {
            var boundary = plane.boundary;
            line.positionCount = boundary.Length;
            for (var i = 0; i < boundary.Length; i++) line.SetPosition(i, new Vector3(boundary[i].x, .005f, boundary[i].y));
        }
        void OnDestroy() { if (material != null) Destroy(material); }
    }
}
