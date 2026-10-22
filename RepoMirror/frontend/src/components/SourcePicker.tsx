import { useEffect, useRef, useState } from "react";
import { listSourceBranches, listSourceCommits } from "../api";
import type { BranchSummary, CommitSummary, RepositorySummary, SourceMode, SourceSelection } from "../types";
import { BranchIcon } from "./Icons";

interface SourcePickerProps {
  selection: SourceSelection;
  repository: RepositorySummary;
  busy: boolean;
  error: string;
  onSelect: (mode: SourceMode, revision: string) => Promise<boolean>;
}

const modeLabels: Record<SourceMode, string> = { working: "工作目录", branch: "分支快照", commit: "单次提交" };

export function SourcePicker(props: SourcePickerProps) {
  const [editingMode, setEditingMode] = useState<"branch" | "commit" | null>(null);
  const { selection, busy, repository, onSelect } = props;
  const refLabel = selection.ref.replace(/^refs\/(heads|remotes)\//, "");

  function chooseMode(mode: SourceMode) {
    if (mode === "working") {
      void onSelect("working", "");
    } else {
      setEditingMode(mode);
    }
  }

  return (
    <div className="source-toolbar">
      <label className="source-mode">
        <span>来源</span>
        <select aria-label="差异来源" value={selection.mode} disabled={busy || !repository.isGitRepo} onChange={(event) => chooseMode(event.target.value as SourceMode)}>
          {Object.entries(modeLabels).map(([mode, label]) => <option key={mode} value={mode}>{label}</option>)}
        </select>
      </label>
      {selection.mode === "working" ? <span className="source-context">含未提交改动</span> : (
        <>
          <span className="source-revision" title={selection.ref + " · " + selection.commit}>
            {selection.mode === "branch" ? <><BranchIcon className="inline-icon" /><span>{refLabel}</span></> : null}
            <code>{selection.commit.slice(0, 8)}</code>
          </span>
          <button className="source-change" type="button" disabled={busy} onClick={() => setEditingMode(selection.mode as "branch" | "commit")}>更改</button>
          <span className="source-scope">{selection.mode === "commit" ? "仅涉及文件" : "完整快照"}</span>
        </>
      )}
      {editingMode ? <SourceDialog {...props} mode={editingMode} onClose={() => setEditingMode(null)} /> : null}
    </div>
  );
}

function SourceDialog({ mode, selection, repository, busy, error, onSelect, onClose }: SourcePickerProps & { mode: "branch" | "commit"; onClose: () => void }) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const requestVersion = useRef(0);
  const [branches, setBranches] = useState<BranchSummary[]>([]);
  const [branch, setBranch] = useState("");
  const [commits, setCommits] = useState<CommitSummary[]>([]);
  const [historyTip, setHistoryTip] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [revision, setRevision] = useState(mode === "commit" && selection.mode === "commit" ? selection.commit : "");
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [localError, setLocalError] = useState("");
  const [attempted, setAttempted] = useState(false);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    const dialog = dialogRef.current!;
    dialog.showModal();
    return () => dialog.close();
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setLocalError("");
    void listSourceBranches().then((items) => {
      if (cancelled) return;
      setBranches(items);
      const active = items.find((item) => item.ref === selection.ref)
        ?? items.find((item) => item.ref === "refs/heads/" + repository.branch);
      setBranch(active?.ref ?? (mode === "branch" ? items[0]?.ref ?? "" : "HEAD"));
      if (mode === "branch" && items.length === 0) setLoading(false);
    }).catch((reason) => {
      if (!cancelled) { setLocalError(String(reason)); setLoading(false); }
    });
    return () => { cancelled = true; };
  }, [retry]);

  useEffect(() => {
    if (!branch) return;
    const version = ++requestVersion.current;
    setLoading(true);
    setLoadingMore(false);
    setLocalError("");
    setCommits([]);
    setHasMore(false);
    void listSourceCommits(branch, 0).then((page) => {
      if (version !== requestVersion.current) return;
      setCommits(page.commits);
      setHistoryTip(page.commits[0]?.hash ?? "");
      setHasMore(page.hasMore);
    }).catch((reason) => {
      if (version === requestVersion.current) setLocalError(String(reason));
    }).finally(() => {
      if (version === requestVersion.current) setLoading(false);
    });
    return () => { requestVersion.current++; };
  }, [branch, retry]);

  async function loadMore() {
    const version = requestVersion.current;
    setLoadingMore(true);
    setLocalError("");
    try {
      const page = await listSourceCommits(historyTip, commits.length);
      if (version === requestVersion.current) {
        setCommits((current) => [...current, ...page.commits]);
        setHasMore(page.hasMore);
      }
    } catch (reason) {
      if (version === requestVersion.current) setLocalError(String(reason));
    } finally {
      if (version === requestVersion.current) setLoadingMore(false);
    }
  }

  async function apply() {
    setAttempted(true);
    if (await onSelect(mode, mode === "branch" ? branch : revision.trim())) onClose();
  }

  const selected = commits.find((commit) => commit.hash === revision);
  const shownError = localError || (attempted ? error : "");
  const canApply = mode === "branch" ? Boolean(branch && historyTip && !loading && !localError) : Boolean(revision.trim());

  return (
    <dialog className="source-dialog" ref={dialogRef} aria-labelledby="source-dialog-title" onCancel={(event) => { event.preventDefault(); if (!busy) onClose(); }}>
      <form onSubmit={(event) => { event.preventDefault(); if (canApply && !busy) void apply(); }}>
        <header className="source-dialog-header">
          <div><h2 id="source-dialog-title">{mode === "commit" ? "选择提交" : "选择分支快照"}</h2><span>源仓库 {repository.slot} · {repository.name}</span></div>
          <button className="icon-button" aria-label="关闭版本选择" type="button" disabled={busy} onClick={onClose}>×</button>
        </header>
        <div className="source-dialog-controls">
          <label className="source-field"><span>分支</span><select autoFocus value={branch} disabled={busy || branches.length === 0} onChange={(event) => { setBranch(event.target.value); setRevision(""); setAttempted(false); }}>
            {mode === "commit" ? <option value="HEAD">当前 HEAD</option> : null}
            {mode === "branch" && branches.length === 0 ? <option value="">暂无分支</option> : null}
            {branches.map((item) => <option key={item.ref} value={item.ref}>{item.name}</option>)}
          </select></label>
          {mode === "commit" ? <label className="source-field source-hash-field"><span>提交号</span><input aria-label="提交号" placeholder="选择下方提交或输入提交号" value={revision} disabled={busy} onChange={(event) => { setRevision(event.target.value); setAttempted(false); }} /></label> : null}
        </div>
        {shownError ? <div className="source-error" role="alert"><span>{shownError}</span>{localError ? <button type="button" className="source-change" disabled={busy || loading} onClick={() => setRetry((value) => value + 1)}>重试</button> : null}</div> : null}
        <div className={`source-history ${mode === "branch" ? "branch-preview" : ""}`} aria-busy={loading || loadingMore}>
          {loading ? <div className="source-empty" role="status">正在读取提交…</div> : commits.length === 0 ? <div className="source-empty">暂无可选提交，可返回工作目录模式。</div> : (
            <>
              {(mode === "commit" ? commits : commits.slice(0, 1)).map((commit) => (
                <button className={`source-commit ${revision === commit.hash ? "selected" : ""}`} key={commit.hash} type="button" disabled={busy || mode === "branch"} aria-pressed={revision === commit.hash} onClick={() => { setRevision(commit.hash); setAttempted(false); }}>
                  <span className="source-commit-subject">{commit.subject}</span>
                  <span className="source-commit-meta"><code>{commit.hash.slice(0, 8)}</code><span>{commit.author}</span><time dateTime={commit.date}>{commit.date.slice(0, 10)}</time></span>
                </button>
              ))}
              {mode === "commit" && hasMore ? <button className="source-load-more" type="button" disabled={busy || loadingMore} onClick={() => void loadMore()}>{loadingMore ? "正在读取…" : "加载更早的提交"}</button> : null}
            </>
          )}
        </div>
        <div className="source-selection-status" title={selected?.subject || revision}>
          {mode === "commit" ? (revision ? `已选 ${revision.slice(0, 12)}${selected ? " · " + selected.subject : ""}` : "尚未选择提交") : "使用分支最新提交，选定后固定版本"}
        </div>
        <p className="source-risk">{mode === "commit" ? "仅同步此提交涉及的文件，修改会覆盖整个文件；合并提交相对第一父提交。" : "按完整快照同步，可能删除目标独有文件；不会切换源仓库分支。"}</p>
        <footer className="source-dialog-footer">
          <button className="secondary-button" type="button" disabled={busy} onClick={onClose}>取消</button>
          <button className="sync-button" type="submit" disabled={busy || !canApply}>{busy ? "正在计算差异…" : mode === "commit" ? "使用此提交" : "使用分支快照"}</button>
        </footer>
      </form>
    </dialog>
  );
}
