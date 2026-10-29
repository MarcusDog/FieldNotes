using System;
using System.IO;
using UnityEditor;
using UnityEditor.Build.Reporting;
using UnityEditor.SceneManagement;
using UnityEditor.XR.Management;
using UnityEditor.XR.Management.Metadata;
using UnityEngine;
using UnityEngine.XR.Management;

namespace FieldNotes.Editor
{
    public static class ProjectSetup
    {
        const string ScenePath = "Assets/FieldNotes/Scenes/Inspection.unity";
        const string ResourcesPath = "Assets/FieldNotes/Resources";

        [MenuItem("FieldNotes/1. Configure project and scene")]
        public static void Configure()
        {
            Directory.CreateDirectory("Assets/FieldNotes/Scenes");
            Directory.CreateDirectory(ResourcesPath);
            AssetDatabase.Refresh();
            // These serialized materials preserve shader references in player builds.
            if (AssetDatabase.LoadAssetAtPath<Material>(ResourcesPath + "/Marker.mat") == null)
            {
                var material = new Material(Shader.Find("Unlit/Color")) { color = new Color(.2f, .65f, .48f) };
                AssetDatabase.CreateAsset(material, ResourcesPath + "/Marker.mat");
            }
            if (AssetDatabase.LoadAssetAtPath<Material>(ResourcesPath + "/Plane.mat") == null)
                AssetDatabase.CreateAsset(new Material(Shader.Find("Sprites/Default")), ResourcesPath + "/Plane.mat");

            PlayerSettings.companyName = "FieldNotes";
            PlayerSettings.productName = "FieldNotes";
            PlayerSettings.insecureHttpOption = InsecureHttpOption.DevelopmentOnly;
            PlayerSettings.SetApplicationIdentifier(BuildTargetGroup.iOS, "com.fieldnotes.inspection");
            PlayerSettings.iOS.targetOSVersionString = "15.0";
            PlayerSettings.iOS.cameraUsageDescription = "使用相机识别巡检现场平面并放置问题标记。";
            PlayerSettings.iOS.appleEnableAutomaticSigning = true;
            PlayerSettings.SetScriptingBackend(UnityEditor.Build.NamedBuildTarget.iOS, ScriptingImplementation.IL2CPP);
            PlayerSettings.defaultInterfaceOrientation = UIOrientation.Portrait;
            PlayerSettings.allowedAutorotateToLandscapeLeft = false;
            PlayerSettings.allowedAutorotateToLandscapeRight = false;
            PlayerSettings.allowedAutorotateToPortraitUpsideDown = false;

            // Legacy touch/uGUI plus Input System pose tracking need Both. Unity
            // may request an Editor restart after this setting changes.
            var settings = new SerializedObject(AssetDatabase.LoadAllAssetsAtPath("ProjectSettings/ProjectSettings.asset")[0]);
            var input = settings.FindProperty("activeInputHandler");
            if (input != null) { input.intValue = 2; settings.ApplyModifiedPropertiesWithoutUndo(); }

            ConfigureXR();
            if (!File.Exists(ScenePath))
            {
                var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
                new GameObject("FieldNotes Bootstrap").AddComponent<Bootstrap>();
                EditorSceneManager.SaveScene(scene, ScenePath);
            }
            EditorBuildSettings.scenes = new[] { new EditorBuildSettingsScene(ScenePath, true) };
            AssetDatabase.SaveAssets();
            Debug.Log("FieldNotes configured. Review XR Plug-in Management > iOS > Project Validation, then export iOS.");
        }

        static void ConfigureXR()
        {
            const string path = "Assets/FieldNotes/XRSettings.asset";
            if (!EditorBuildSettings.TryGetConfigObject(XRGeneralSettings.k_SettingsKey, out XRGeneralSettingsPerBuildTarget perTarget))
            {
                perTarget = ScriptableObject.CreateInstance<XRGeneralSettingsPerBuildTarget>();
                AssetDatabase.CreateAsset(perTarget, path);
                EditorBuildSettings.AddConfigObject(XRGeneralSettings.k_SettingsKey, perTarget, true);
            }
            var general = perTarget.SettingsForBuildTarget(BuildTargetGroup.iOS);
            if (general == null)
            {
                perTarget.CreateDefaultSettingsForBuildTarget(BuildTargetGroup.iOS);
                general = perTarget.SettingsForBuildTarget(BuildTargetGroup.iOS);
            }
            if (general.Manager == null) perTarget.CreateDefaultManagerSettingsForBuildTarget(BuildTargetGroup.iOS);
            general.InitManagerOnStart = true;
            if (!XRPackageMetadataStore.AssignLoader(general.Manager, "UnityEngine.XR.ARKit.ARKitLoader", BuildTargetGroup.iOS))
                throw new InvalidOperationException("Cannot enable ARKit. Install iOS Build Support and enable ARKit under XR Plug-in Management.");
            EditorUtility.SetDirty(perTarget);
            EditorUtility.SetDirty(general);
            EditorUtility.SetDirty(general.Manager);
        }

        [MenuItem("FieldNotes/2. Export iOS Xcode project")]
        public static void ExportIOS()
        {
            Configure();
            if (!BuildPipeline.IsBuildTargetSupported(BuildTargetGroup.iOS, BuildTarget.iOS))
                throw new InvalidOperationException("Install Unity iOS Build Support through Unity Hub first.");
            var location = Environment.GetEnvironmentVariable("FIELDNOTES_IOS_OUTPUT") ?? "Builds/iOS";
            var report = BuildPipeline.BuildPlayer(new BuildPlayerOptions
            {
                scenes = new[] { ScenePath }, locationPathName = location,
                target = BuildTarget.iOS, options = BuildOptions.Development
            });
            if (report.summary.result != BuildResult.Succeeded) throw new InvalidOperationException("iOS export failed: " + report.summary.result);
        }
    }
}
