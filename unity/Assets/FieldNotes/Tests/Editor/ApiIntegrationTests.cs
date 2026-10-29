using System;
using System.Collections;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;

namespace FieldNotes.Tests
{
    public class ApiIntegrationTests
    {
        [UnityTest]
        public IEnumerator RealGoTransportPreservesPayloadAndDeduplicatesRetry()
        {
            var endpoint = Environment.GetEnvironmentVariable("FIELDNOTES_TEST_API");
            if (string.IsNullOrEmpty(endpoint)) Assert.Ignore("Set FIELDNOTES_TEST_API to an isolated loopback Go test service to run this integration test.");
            Assert.That(Endpoint.TryNormalize(endpoint, out endpoint), Is.True);
            Assert.That(new Uri(endpoint).IsLoopback, Is.True, "Integration fixtures may only be sent to an explicitly configured local test service.");

            string error = null;
            yield return AwaitRequest(IssueApi.Health(endpoint, value => error = value));
            Assert.That(error, Is.Null);

            var payload = CreateIssue.From("〔Unity Editor API 验收〕中文巡检记录", "显式传输测试，不是真机 AR 上报。", "high", new Vector3(.42f, .03f, 1.26f));
            var json = JsonUtility.ToJson(payload);
            var key = Guid.NewGuid().ToString();
            Issue first = null;
            yield return AwaitRequest(IssueApi.Create(endpoint, json, key, (issue, failure) => { first = issue; error = failure; }));
            Assert.That(error, Is.Null);
            Assert.That(first, Is.Not.Null);
            Assert.That(first.id, Is.Not.Empty);
            Assert.That(first.title, Is.EqualTo(payload.title));
            Assert.That(first.status, Is.EqualTo("open"));
            Assert.That(first.position.x, Is.EqualTo(.42f).Within(.0001f));
            Assert.That(first.createdAt, Is.Not.Empty);

            Issue retry = null;
            yield return AwaitRequest(IssueApi.Create(endpoint, json, key, (issue, failure) => { retry = issue; error = failure; }));
            Assert.That(error, Is.Null);
            Assert.That(retry.id, Is.EqualTo(first.id));
        }

        static IEnumerator AwaitRequest(IEnumerator request)
        {
            // EditMode's iterator runner doesn't wait for the player's native
            // AsyncOperation yield. Drive that wait explicitly in this harness.
            // Runtime coroutines still use Unity's normal AsyncOperation yield.
            try
            {
                while (request.MoveNext())
                {
                    if (request.Current is AsyncOperation operation)
                    {
                        while (!operation.isDone) yield return null;
                    }
                    else yield return request.Current;
                }
            }
            finally { (request as IDisposable)?.Dispose(); }
        }
    }
}
