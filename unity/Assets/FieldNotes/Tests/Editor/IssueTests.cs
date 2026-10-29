using NUnit.Framework;
using UnityEngine;

namespace FieldNotes.Tests
{
    public class IssueTests
    {
        [Test]
        public void PlaneTemplateKeepsClonesActiveWithoutRenderingTheTemplate()
        {
            var template = Bootstrap.CreatePlaneTemplate();
            try
            {
                Assert.That(template.activeSelf, Is.True, "ARTrackableManager copies activeSelf to every detected plane");
                Assert.That(template.activeInHierarchy, Is.False, "The source template must stay hidden");
            }
            finally { Object.DestroyImmediate(template.transform.parent.gameObject); }
        }

        [Test]
        public void EndpointAcceptsLanAndRemovesTrailingSlash()
        {
            Assert.That(Endpoint.TryNormalize(" http://192.168.1.20:8080/ ", out var url), Is.True);
            Assert.That(url, Is.EqualTo("http://192.168.1.20:8080"));
        }

        [TestCase("192.168.1.4:8090")]
        [TestCase("http://192.168.1.4:8090")]
        [TestCase("  http://192.168.1.4:8090/\n")]
        [TestCase("http：／／192.168.1.4：8090")]
        [TestCase("\u200bhttp://192.168.1.4:8090\ufeff")]
        public void PhoneAddressEntryNormalizesCommonInputForms(string input)
        {
            Assert.That(Endpoint.TryNormalize(input, out var value), Is.True);
            Assert.That(value, Is.EqualTo("http://192.168.1.4:8090"));
        }

        [TestCase("file:///etc/passwd")]
        [TestCase("http://user:pass@example.com")]
        [TestCase("http://localhost:8080/api")]
        [TestCase("https://example.com/?token=secret")]
        [TestCase("not a url")]
        public void EndpointRejectsUnsupportedOrAmbiguousValues(string input)
        {
            Assert.That(Endpoint.TryNormalize(input, out _), Is.False);
        }

        [TestCase(true, false, false, true, true, false)]
        [TestCase(false, true, false, true, true, false)]
        [TestCase(false, false, true, true, true, false)]
        [TestCase(false, false, false, false, true, false)]
        [TestCase(false, false, false, true, false, false)]
        [TestCase(false, false, false, true, true, true)]
        public void PlacementRequiresFreshTouchTrackingAndNoUI(bool overUI, bool busy, bool locked, bool tracking, bool began, bool expected)
        {
            Assert.That(PlacementPolicy.CanPlace(overUI, busy, locked, tracking, began), Is.EqualTo(expected));
        }

        [Test]
        public void PayloadSerializesContractAndNoServerFields()
        {
            var draft = CreateIssue.From("  入口墙面破损  ", "裂缝", "high", new Vector3(.42f, .03f, 1.26f));
            var json = JsonUtility.ToJson(draft);
            StringAssert.Contains("\"title\":\"入口墙面破损\"", json);
            StringAssert.Contains("\"position\":", json);
            var decoded = JsonUtility.FromJson<CreateIssue>(json);
            Assert.That(decoded.position.x, Is.EqualTo(.42f).Within(.00001f));
            StringAssert.DoesNotContain("createdAt", json);
            Assert.That(draft.ValidationError(), Is.Null);
        }

        [Test]
        public void ValidationRejectsBlankTitleEnumAndNonfinitePosition()
        {
            Assert.That(CreateIssue.From("  ", "", "medium", Vector3.zero).ValidationError(), Is.Not.Null);
            Assert.That(CreateIssue.From("x", "", "urgent", Vector3.zero).ValidationError(), Is.Not.Null);
            Assert.That(CreateIssue.From("x", "", "low", new Vector3(float.NaN, 0, 0)).ValidationError(), Is.Not.Null);
        }
    }
}
