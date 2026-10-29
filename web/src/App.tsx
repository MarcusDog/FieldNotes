import { useCallback, useEffect, useRef, useState } from "react";
import type { Issue, Priority, Status } from "./api";
import {
  listIssues,
  priorities,
  priorityLabels,
  statuses,
  statusLabels,
  updateStatus,
} from "./api";
import "./styles.css";

const dateTime = (value: string) =>
  new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(value));
const fullDateTime = (value: string) =>
  new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(new Date(value));
const errorMessage = (error: unknown) =>
  error instanceof Error ? error.message : "请求未完成，请稍后重试。";
function ArrowIcon() {
  return (
    <svg
      width="17"
      height="17"
      viewBox="0 0 20 20"
      fill="none"
      aria-hidden="true"
    >
      <path d="M4 10h12m-5-5 5 5-5 5" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  );
}
function RefreshIcon({ active }: { active: boolean }) {
  return (
    <svg
      className={active ? "refresh-icon spinning" : "refresh-icon"}
      width="16"
      height="16"
      viewBox="0 0 20 20"
      fill="none"
      aria-hidden="true"
    >
      <path
        d="M16 7a6.5 6.5 0 1 0 .4 5M16 3v4h-4"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
function StatusTag({ status }: { status: Status }) {
  return (
    <span className={`status-tag status-${status}`}>
      <span className="status-dot" />
      {statusLabels[status]}
    </span>
  );
}
function Inspector({
  issue,
  pending,
  refreshing,
  error,
  onSave,
  onClose,
}: {
  issue: Issue;
  pending: boolean;
  refreshing: boolean;
  error: string | null;
  onSave: (status: Status) => Promise<boolean>;
  onClose: () => void;
}) {
  const [draft, setDraft] = useState<Status>(issue.status);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    setDraft(issue.status);
  }, [issue.status, issue.id]);
  useEffect(() => {
    heading.current?.focus();
  }, [issue.id]);
  async function save() {
    if (!(await onSave(draft))) setDraft(issue.status);
  }
  return (
    <section
      className="inspector"
      role="region"
      aria-label="问题详情"
      aria-busy={pending || refreshing}
    >
      <div className="inspector-top">
        <span className="eyebrow">问题详情</span>
        <button
          className="icon-button"
          aria-label="关闭详情"
          onClick={onClose}
          disabled={pending}
        >
          ×
        </button>
      </div>
      <div className="inspector-content">
        <div className="detail-meta">
          <StatusTag status={issue.status} />
          <span className="priority-detail">
            {priorityLabels[issue.priority]}优先级
          </span>
        </div>
        <h2 ref={heading} tabIndex={-1}>
          {issue.title}
        </h2>
        <p className="issue-id" title={issue.id}>
          ID / {issue.id}
        </p>
        <div className="detail-section">
          <h3>现场描述</h3>
          <p className={`description ${issue.description ? "" : "muted"}`}>
            {issue.description || "上报时未填写描述。"}
          </p>
        </div>
        <div className="detail-section">
          <h3>
            标记位置 <span>米 / m</span>
          </h3>
          <dl className="coordinates">
            {(["x", "y", "z"] as const).map((axis) => (
              <div key={axis}>
                <dt>{axis.toUpperCase()}</dt>
                <dd>{issue.position[axis].toFixed(3)}</dd>
              </div>
            ))}
          </dl>
          <p className="coordinate-help">
            本次 AR 会话中的世界坐标，不是 GPS。
          </p>
        </div>
        <div className="detail-section">
          <h3>记录时间</h3>
          <dl className="timestamps">
            <div>
              <dt>创建</dt>
              <dd>{fullDateTime(issue.createdAt)}</dd>
            </div>
            <div>
              <dt>更新</dt>
              <dd>{fullDateTime(issue.updatedAt)}</dd>
            </div>
          </dl>
        </div>
      </div>
      <form
        className="status-form"
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        <label htmlFor="issue-status">问题状态</label>
        <div className="status-form-actions">
          <select
            id="issue-status"
            value={draft}
            onChange={(event) => setDraft(event.target.value as Status)}
            disabled={pending || refreshing}
          >
            {statuses.map((status) => (
              <option key={status} value={status}>
                {statusLabels[status]}
              </option>
            ))}
          </select>
          <button
            className="primary-button"
            type="submit"
            disabled={pending || refreshing || draft === issue.status}
          >
            {pending ? "正在保存…" : "保存状态"}
          </button>
        </div>
        {error && (
          <div className="update-error" role="alert">
            <strong>状态未保存</strong>
            <p>{error}</p>
            <p>原状态已保留，请重新选择后重试。</p>
          </div>
        )}
      </form>
    </section>
  );
}
export default function App() {
  const [issues, setIssues] = useState<Issue[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<Status | "all">("all");
  const [priorityFilter, setPriorityFilter] = useState<Priority | "all">("all");
  const [query, setQuery] = useState("");
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [patchError, setPatchError] = useState<string | null>(null);
  const [notice, setNotice] = useState("");
  const [lastLoaded, setLastLoaded] = useState<Date | null>(null);
  const loadSequence = useRef(0);
  const updatePending = useRef(false);
  const load = useCallback(async () => {
    const sequence = ++loadSequence.current;
    setLoading(true);
    setLoadError(null);
    setNotice("");
    try {
      const result = await listIssues();
      if (sequence !== loadSequence.current) return;
      setIssues(result);
      setLastLoaded(new Date());
      setSelectedId((id) =>
        id && result.some((issue) => issue.id === id) ? id : null,
      );
      setPatchError(null);
    } catch (error) {
      if (sequence === loadSequence.current) setLoadError(errorMessage(error));
    } finally {
      if (sequence === loadSequence.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    void load();
    return () => {
      loadSequence.current++;
    };
  }, [load]);
  const selected = issues?.find((issue) => issue.id === selectedId);
  const total = issues?.length ?? 0;
  const resolved =
    issues?.filter((issue) => issue.status === "resolved").length ?? 0;
  const inProgress =
    issues?.filter((issue) => issue.status === "in_progress").length ?? 0;
  const open = issues?.filter((issue) => issue.status === "open").length ?? 0;
  const search = query.trim().toLocaleLowerCase();
  const filtered =
    issues?.filter(
      (issue) =>
        (statusFilter === "all" || issue.status === statusFilter) &&
        (priorityFilter === "all" || issue.priority === priorityFilter) &&
        (!search ||
          `${issue.title} ${issue.description} ${issue.id}`
            .toLocaleLowerCase()
            .includes(search)),
    ) ?? [];
  const hasFilters =
    statusFilter !== "all" || priorityFilter !== "all" || !!query;
  function resetFilters() {
    setStatusFilter("all");
    setPriorityFilter("all");
    setQuery("");
  }
  function closeInspector() {
    const id = selectedId;
    setSelectedId(null);
    setPatchError(null);
    const returnTarget = id ? document.getElementById(`issue-${id}`) : null;
    (returnTarget ?? document.getElementById("issue-search"))?.focus();
  }
  async function saveStatus(status: Status): Promise<boolean> {
    if (!selected || loading || updatePending.current) return false;
    const id = selected.id;
    updatePending.current = true;
    setPendingId(id);
    setPatchError(null);
    setNotice("");
    try {
      const updated = await updateStatus(id, status);
      setIssues(
        (current) =>
          current?.map((issue) => (issue.id === id ? updated : issue)) ?? null,
      );
      setNotice("状态已更新");
      return true;
    } catch (error) {
      setPatchError(errorMessage(error));
      return false;
    } finally {
      updatePending.current = false;
      setPendingId(null);
    }
  }
  return (
    <div className="app-shell">
      <a href="#workspace" className="skip-link">
        跳转到问题列表
      </a>
      <aside className="sidebar" aria-label="工作区导航">
        <a className="brand" href="#workspace" aria-label="FieldNotes 工作台">
          <span className="brand-mark" aria-hidden="true">
            <span />
            <span />
            <span />
          </span>
          <span>
            FieldNotes<span className="brand-subtitle">现场巡检</span>
          </span>
        </a>
        <div className="nav-section-label">工作空间</div>
        <nav>
          <a className="nav-item active" href="#workspace" aria-current="page">
            <svg
              width="19"
              height="19"
              viewBox="0 0 20 20"
              fill="none"
              aria-hidden="true"
            >
              <path
                d="M5 3h10v14H5zM8 7h4M8 10h4M8 13h2"
                stroke="currentColor"
                strokeWidth="1.3"
              />
            </svg>
            <span>巡检问题</span>
            <span className="nav-count">{issues ? total : "—"}</span>
          </a>
        </nav>
        <div className="sidebar-bottom">
          <span className="sidebar-rule" />
          <p>
            从现场记录
            <br />
            到问题解决。
          </p>
          <span className="edition">FIELD OPERATIONS / 01</span>
        </div>
      </aside>
      <main id="workspace" className="workspace" tabIndex={-1}>
        <header className="workspace-top">
          <div className="breadcrumb">
            工作空间 <span>/</span> 问题管理
          </div>
          <span className="workspace-label">巡检工作台</span>
        </header>
        <div className="page-heading">
          <div>
            <p className="eyebrow">ISSUE REGISTER</p>
            <h1>
              巡检问题<span className="heading-period">.</span>
            </h1>
            <p className="page-description">查看现场记录，跟进每一个问题。</p>
          </div>
          <button
            className="secondary-button refresh-button"
            onClick={() => void load()}
            disabled={loading || !!pendingId}
            aria-label="刷新列表"
          >
            <RefreshIcon active={loading} />
            {loading && issues ? "正在刷新" : "刷新列表"}
          </button>
        </div>
        <div className="summary" aria-label="问题统计">
          <div>
            <span className="summary-number">
              {issues ? total.toString().padStart(2, "0") : "—"}
            </span>
            <span>全部问题</span>
          </div>
          <div>
            <span className="summary-number">
              {issues ? open.toString().padStart(2, "0") : "—"}
            </span>
            <span>待处理</span>
          </div>
          <div>
            <span className="summary-number">
              {issues ? inProgress.toString().padStart(2, "0") : "—"}
            </span>
            <span>处理中</span>
          </div>
          <div>
            <span className="summary-number resolved-number">
              {issues ? resolved.toString().padStart(2, "0") : "—"}
            </span>
            <span>已解决</span>
          </div>
        </div>
        <div className={`work-area ${selected ? "has-inspector" : ""}`}>
          <section
            className="register"
            aria-label="问题列表"
            aria-busy={loading}
          >
            <div className="filter-toolbar">
              <div className="search-control">
                <svg
                  width="17"
                  height="17"
                  viewBox="0 0 20 20"
                  fill="none"
                  aria-hidden="true"
                >
                  <circle
                    cx="8.5"
                    cy="8.5"
                    r="5.5"
                    stroke="currentColor"
                    strokeWidth="1.4"
                  />
                  <path
                    d="m13 13 4 4"
                    stroke="currentColor"
                    strokeWidth="1.4"
                  />
                </svg>
                <input
                  type="search"
                  id="issue-search"
                  aria-label="搜索问题"
                  placeholder="搜索标题、描述或 ID"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
              <div className="filters">
                <label className="sr-only" htmlFor="filter-status">
                  筛选状态
                </label>
                <select
                  id="filter-status"
                  value={statusFilter}
                  onChange={(event) =>
                    setStatusFilter(event.target.value as Status | "all")
                  }
                >
                  <option value="all">全部状态</option>
                  {statuses.map((status) => (
                    <option key={status} value={status}>
                      {statusLabels[status]}
                    </option>
                  ))}
                </select>
                <label className="sr-only" htmlFor="filter-priority">
                  筛选优先级
                </label>
                <select
                  id="filter-priority"
                  value={priorityFilter}
                  onChange={(event) =>
                    setPriorityFilter(event.target.value as Priority | "all")
                  }
                >
                  <option value="all">全部优先级</option>
                  {[...priorities].reverse().map((priority) => (
                    <option key={priority} value={priority}>
                      {priorityLabels[priority]}优先级
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="list-meta">
              <span>
                {issues
                  ? `显示 ${filtered.length} / ${total} 条记录`
                  : "现场问题记录"}
                {hasFilters && filtered.length > 0 && (
                  <button
                    className="text-button clear-filters"
                    onClick={resetFilters}
                  >
                    清除筛选
                  </button>
                )}
              </span>
              <span>
                {lastLoaded
                  ? `更新于 ${lastLoaded.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", hour12: false })}`
                  : "等待同步"}
              </span>
            </div>
            {loadError && (
              <div className="load-error" role="alert">
                <div>
                  <strong>
                    {issues
                      ? "刷新未完成，当前显示上次加载的记录"
                      : "暂时无法加载问题"}
                  </strong>
                  <p>{loadError}</p>
                </div>
                <button
                  className="secondary-button"
                  onClick={() => void load()}
                  disabled={loading}
                >
                  重试加载
                </button>
              </div>
            )}
            {loading && !issues && (
              <div className="state-placeholder" role="status">
                <span className="loading-bar" />
                <h2>正在加载巡检问题</h2>
                <p>正在连接现场记录服务…</p>
              </div>
            )}
            {issues && total === 0 && (
              <div className="state-placeholder">
                <div className="empty-symbol" aria-hidden="true">
                  ＋
                </div>
                <h2>暂无巡检问题</h2>
                <p>在手机端完成现场标注并上报后，问题会显示在这里。</p>
                <button
                  className="text-button"
                  onClick={() => void load()}
                  disabled={loading}
                >
                  刷新查看 <ArrowIcon />
                </button>
              </div>
            )}
            {issues && total > 0 && filtered.length === 0 && (
              <div className="state-placeholder">
                <div className="empty-symbol" aria-hidden="true">
                  ∅
                </div>
                <h2>没有符合条件的问题</h2>
                <p>试试其他关键词，或清除筛选查看全部记录。</p>
                <button className="text-button" onClick={resetFilters}>
                  清除筛选 <ArrowIcon />
                </button>
              </div>
            )}
            {filtered.length > 0 && (
              <div className="table-scroll">
                <table>
                  <caption className="sr-only">
                    巡检问题列表，选择标题查看和处理问题
                  </caption>
                  <thead>
                    <tr>
                      <th className="index-column" scope="col">
                        序号
                      </th>
                      <th className="title-heading" scope="col">
                        问题标题
                      </th>
                      <th className="priority-heading" scope="col">
                        优先级
                      </th>
                      <th className="status-heading" scope="col">
                        状态
                      </th>
                      <th className="date-column" scope="col">
                        上报时间
                      </th>
                      <th className="arrow-column">
                        <span className="sr-only">详情</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((issue, index) => (
                      <tr
                        key={issue.id}
                        className={
                          issue.id === selectedId ? "selected-row" : ""
                        }
                      >
                        <td className="index-column row-index">
                          {(index + 1).toString().padStart(2, "0")}
                        </td>
                        <td className="title-cell">
                          <button
                            id={`issue-${issue.id}`}
                            className="issue-button"
                            aria-label={`查看问题：${issue.title}`}
                            aria-expanded={issue.id === selectedId}
                            disabled={!!pendingId || loading}
                            onClick={() => {
                              setSelectedId(issue.id);
                              setPatchError(null);
                              setNotice("");
                            }}
                          >
                            {issue.title}
                          </button>
                          <span className="table-id">{issue.id}</span>
                        </td>
                        <td>
                          <span
                            className={`priority priority-${issue.priority}`}
                          >
                            <span className="priority-lines" aria-hidden="true">
                              <i />
                              <i />
                              <i />
                            </span>
                            {priorityLabels[issue.priority]}
                          </span>
                        </td>
                        <td>
                          <StatusTag status={issue.status} />
                        </td>
                        <td className="date-column table-date">
                          <time dateTime={issue.createdAt}>
                            {dateTime(issue.createdAt)}
                          </time>
                        </td>
                        <td className="arrow-column">
                          <ArrowIcon />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {issues && total > 0 && (
              <footer className="register-footer">
                <span>记录来自手机端现场上报</span>
                <span>共 {total} 条</span>
              </footer>
            )}
          </section>
          {selected && (
            <Inspector
              key={selected.id}
              issue={selected}
              pending={pendingId === selected.id}
              refreshing={loading}
              error={patchError}
              onSave={saveStatus}
              onClose={closeInspector}
            />
          )}
        </div>
        <p
          className={`save-notice ${notice ? "visible" : ""}`}
          role={notice ? "status" : undefined}
          aria-live="polite"
        >
          {notice}
        </p>
        <footer className="page-footer">
          <span>FieldNotes</span>
          <span>现场巡检 · 问题管理</span>
        </footer>
      </main>
    </div>
  );
}
