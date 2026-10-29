export const priorities = ["low", "medium", "high"] as const;
export const statuses = ["open", "in_progress", "resolved"] as const;
export type Priority = (typeof priorities)[number];
export type Status = (typeof statuses)[number];
export interface Issue {
  id: string;
  title: string;
  description: string;
  priority: Priority;
  status: Status;
  position: { x: number; y: number; z: number };
  createdAt: string;
  updatedAt: string;
}
export const priorityLabels: Record<Priority, string> = {
  low: "低",
  medium: "中",
  high: "高",
};
export const statusLabels: Record<Status, string> = {
  open: "待处理",
  in_progress: "处理中",
  resolved: "已解决",
};
export const REQUEST_TIMEOUT_MS = 10_000;
const record = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
export function isIssue(value: unknown): value is Issue {
  if (!record(value) || !record(value.position)) return false;
  return (
    typeof value.id === "string" &&
    value.id.length > 0 &&
    typeof value.title === "string" &&
    value.title.trim().length > 0 &&
    typeof value.description === "string" &&
    priorities.includes(value.priority as Priority) &&
    statuses.includes(value.status as Status) &&
    ["x", "y", "z"].every(
      (key) =>
        typeof value.position === "object" &&
        Number.isFinite((value.position as Record<string, unknown>)[key]),
    ) &&
    [value.createdAt, value.updatedAt].every(
      (date) => typeof date === "string" && Number.isFinite(Date.parse(date)),
    )
  );
}
async function request(path: string, init: RequestInit = {}): Promise<unknown> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const response = await fetch(path, { ...init, signal: controller.signal });
    let body: unknown;
    try {
      body = await response.json();
    } catch {
      throw new Error("服务返回的数据格式异常，请重试或检查后端服务。");
    }
    if (!response.ok) {
      const message =
        record(body) &&
        record(body.error) &&
        typeof body.error.message === "string"
          ? body.error.message
          : `请求失败（HTTP ${response.status}）`;
      throw new Error(message);
    }
    if (!record(body) || !("data" in body))
      throw new Error("服务返回的数据格式异常，请重试或检查后端服务。");
    return body.data;
  } catch (error) {
    if (controller.signal.aborted)
      throw new Error("巡检服务响应超时，请检查网络连接后重试。");
    if (error instanceof TypeError)
      throw new Error("无法连接巡检服务，请确认后端已启动并检查网络连接。");
    throw error;
  } finally {
    clearTimeout(timeout);
  }
}
export async function listIssues(): Promise<Issue[]> {
  const data = await request("/api/issues");
  if (
    !Array.isArray(data) ||
    !data.every(isIssue) ||
    new Set(data.map((issue) => issue.id)).size !== data.length
  )
    throw new Error("服务返回的数据格式异常，请重试或检查后端服务。");
  return data;
}
export async function updateStatus(id: string, status: Status): Promise<Issue> {
  const data = await request(`/api/issues/${encodeURIComponent(id)}/status`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status }),
  });
  if (!isIssue(data) || data.id !== id || data.status !== status)
    throw new Error("服务返回的数据格式异常，请刷新确认问题的最新状态。");
  return data;
}
