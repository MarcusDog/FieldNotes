import { afterEach, describe, expect, it, vi } from "vitest";
import { listIssues, REQUEST_TIMEOUT_MS } from "./api";
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});
describe("API 防护", () => {
  it("aborts a stalled connection and supplies a retryable timeout message", async () => {
    vi.useFakeTimers();
    vi.stubGlobal(
      "fetch",
      vi.fn(
        (_path: string, init: RequestInit) =>
          new Promise((_resolve, reject) => {
            init.signal?.addEventListener("abort", () =>
              reject(new DOMException("Aborted", "AbortError")),
            );
          }),
      ),
    );
    const request = listIssues();
    const assertion = expect(request).rejects.toThrow("巡检服务响应超时");
    await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS);
    await assertion;
  });
  it("reports the backend error message", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({
              error: { code: "storage_error", message: "记录暂时不可用" },
            }),
            { status: 500 },
          ),
        ),
    );
    await expect(listIssues()).rejects.toThrow("记录暂时不可用");
  });
});
