import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

const issue = {
  id: "issue-001",
  title: "走廊照明需要检修",
  description: "三楼东侧灯具闪烁。",
  priority: "high",
  status: "open",
  position: { x: 1.25, y: 0, z: -3.5 },
  createdAt: "2026-10-29T04:00:00Z",
  updatedAt: "2026-10-29T04:00:00Z",
};
const other = {
  ...issue,
  id: "issue-002",
  title: "窗框密封条松动",
  description: "南侧窗框密封条松动。",
  priority: "low",
  status: "resolved",
};
const fetchMock = vi.fn<typeof fetch>();
function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}
beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
  fetchMock.mockReset();
});

describe("巡检工作台", () => {
  it("loads the real list and opens an accessible inspector with coordinates", async () => {
    fetchMock.mockResolvedValue(json({ data: [issue, other] }));
    render(<App />);
    expect(screen.getByRole("status")).toHaveTextContent("正在加载");
    await userEvent.click(
      await screen.findByRole("button", { name: /查看问题：走廊照明需要检修/ }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/issues",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    const panel = screen.getByRole("region", { name: "问题详情" });
    expect(within(panel).getByText(issue.description)).toBeVisible();
    expect(within(panel).getByText("1.250")).toBeVisible();
    expect(within(panel).getByLabelText("问题状态")).toHaveValue("open");
  });
  it("shows an empty collection without invented records", async () => {
    fetchMock.mockResolvedValue(json({ data: [] }));
    render(<App />);
    expect(await screen.findByText("暂无巡检问题")).toBeVisible();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });
  it("shows service failure and retries successfully", async () => {
    fetchMock
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValueOnce(json({ data: [issue] }));
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "无法连接巡检服务",
    );
    await userEvent.click(screen.getByRole("button", { name: "重试加载" }));
    expect(
      await screen.findByRole("button", { name: /查看问题：走廊照明需要检修/ }),
    ).toBeVisible();
  });
  it("filters by status, priority and search, with a distinct no-match state", async () => {
    fetchMock.mockResolvedValue(json({ data: [issue, other] }));
    render(<App />);
    await screen.findByRole("table");
    await userEvent.selectOptions(
      screen.getByLabelText("筛选状态"),
      "resolved",
    );
    expect(
      screen.queryByRole("button", { name: /查看问题：走廊照明/ }),
    ).not.toBeInTheDocument();
    await userEvent.selectOptions(screen.getByLabelText("筛选优先级"), "high");
    expect(screen.getByText("没有符合条件的问题")).toBeVisible();
    await userEvent.click(screen.getByRole("button", { name: "清除筛选" }));
    await userEvent.type(
      screen.getByRole("searchbox", { name: "搜索问题" }),
      "灯具",
    );
    expect(
      screen.getByRole("button", { name: /查看问题：走廊照明/ }),
    ).toBeVisible();
    expect(
      screen.queryByRole("button", { name: /查看问题：窗框/ }),
    ).not.toBeInTheDocument();
  });
  it("updates status and preserves the selected issue after refresh", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ data: [issue] }))
      .mockResolvedValueOnce(
        json({ data: { ...issue, status: "in_progress" } }),
      )
      .mockResolvedValueOnce(
        json({ data: [{ ...issue, status: "in_progress" }] }),
      );
    render(<App />);
    await userEvent.click(
      await screen.findByRole("button", { name: /查看问题：走廊照明/ }),
    );
    await userEvent.selectOptions(screen.getByLabelText("筛选状态"), "open");
    await userEvent.selectOptions(
      screen.getByLabelText("问题状态"),
      "in_progress",
    );
    await userEvent.click(screen.getByRole("button", { name: "保存状态" }));
    expect(await screen.findByText("状态已更新")).toBeVisible();
    expect(screen.getByText("没有符合条件的问题")).toBeVisible();
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/issues/issue-001/status",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({ status: "in_progress" }),
      }),
    );
    await userEvent.click(screen.getByRole("button", { name: "刷新列表" }));
    await waitFor(() =>
      expect(screen.getByLabelText("问题状态")).toHaveValue("in_progress"),
    );
    expect(screen.getByRole("region", { name: "问题详情" })).toBeVisible();
  });
  it("preserves original state and detail on failed update and prevents duplicate submission", async () => {
    let rejectUpdate: ((error: Error) => void) | undefined;
    fetchMock
      .mockResolvedValueOnce(json({ data: [issue] }))
      .mockImplementationOnce(
        () =>
          new Promise((_resolve, reject) => {
            rejectUpdate = reject;
          }),
      );
    render(<App />);
    await userEvent.click(
      await screen.findByRole("button", { name: /查看问题：走廊照明/ }),
    );
    await userEvent.selectOptions(
      screen.getByLabelText("问题状态"),
      "resolved",
    );
    await userEvent.click(screen.getByRole("button", { name: "保存状态" }));
    expect(screen.getByRole("button", { name: "正在保存…" })).toBeDisabled();
    expect(screen.getByLabelText("问题状态")).toBeDisabled();
    rejectUpdate?.(new TypeError("network down"));
    expect(await screen.findByRole("alert")).toHaveTextContent("状态未保存");
    expect(screen.getByLabelText("问题状态")).toHaveValue("open");
    expect(screen.getByText(issue.description)).toBeVisible();
    expect(screen.getAllByText("待处理").length).toBeGreaterThan(0);
  });
  it("closes a selection removed on refresh and retains old records when refresh fails", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ data: [issue] }))
      .mockRejectedValueOnce(new TypeError("offline"))
      .mockResolvedValueOnce(json({ data: [] }));
    render(<App />);
    await userEvent.click(
      await screen.findByRole("button", { name: /查看问题：走廊照明/ }),
    );
    await userEvent.click(screen.getByRole("button", { name: "刷新列表" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "当前显示上次加载的记录",
    );
    expect(screen.getByRole("region", { name: "问题详情" })).toBeVisible();
    await userEvent.click(screen.getByRole("button", { name: "重试加载" }));
    expect(await screen.findByText("暂无巡检问题")).toBeVisible();
    expect(
      screen.queryByRole("region", { name: "问题详情" }),
    ).not.toBeInTheDocument();
  });
  it("handles unexpected server shapes instead of crashing", async () => {
    fetchMock.mockResolvedValue(json({ data: [{ ...issue, position: null }] }));
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "服务返回的数据格式异常",
    );
  });
});
