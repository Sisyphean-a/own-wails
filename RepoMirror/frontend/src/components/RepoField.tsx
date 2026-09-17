import type { ChangeEvent } from "react";
import type { RepositorySummary, RepositorySlot } from "../types";
import { BranchIcon, FolderIcon } from "./Icons";
import { repoStateLabel, repoStateTone } from "./ui";

interface RepoFieldProps {
  busy: boolean;
  label: string;
  repository: RepositorySummary;
  isTarget: boolean;
  onSelect: (slot: RepositorySlot) => void;
  onSelectWorktree: (slot: RepositorySlot, path: string) => void;
}

export function RepoField({ busy, label, repository, isTarget, onSelect, onSelectWorktree }: RepoFieldProps) {
  const branch = repository.isGitRepo ? repository.branch || "HEAD" : "—";
  const tone = repoStateTone(repository);
  const path = repository.path || "未选择 Git 仓库";
  const selectLabel = repository.isConfigured ? "更换仓库" : "选择仓库";
  const buttonTone = repository.isConfigured ? "configured" : "empty";
  const canSelectWorktree = repository.isGitRepo && repository.worktrees.length > 1;

  const handleWorktreeChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextPath = event.target.value;
    if (!nextPath || nextPath === repository.path) {
      return;
    }
    void onSelectWorktree(repository.slot, nextPath);
  };

  return (
    <div className="repo-row">
      <div className={`repo-slot ${label === "B" ? "secondary" : ""}`}>{label}</div>
      <div className="repo-row-main">
        <div className="repo-row-path" title={path}>
          {path}
        </div>
      </div>
      <div className="repo-inline-meta">
        <BranchIcon className="tiny-icon muted-icon" />
        <span className="repo-branch">{branch}</span>
      </div>
      <div className={`repo-inline-meta ${tone}`}>
        <span className="repo-state-dot" />
        <span className="repo-state-label">{repoStateLabel(repository)}</span>
      </div>
      {canSelectWorktree ? (
        <select
          className="repo-worktree-select"
          disabled={busy}
          onChange={handleWorktreeChange}
          title={repository.worktreeError || "切换工作树"}
          value={repository.path}
        >
          {repository.worktrees.map((worktree) => (
            <option key={worktree.path} value={worktree.path}>
              {worktree.branch} · {worktree.name}
            </option>
          ))}
        </select>
      ) : null}
      {isTarget ? <div className="repo-target-tag">目标</div> : null}
      <button className={`repo-select-button ${buttonTone}`} disabled={busy} onClick={() => onSelect(repository.slot)} type="button">
        <FolderIcon className="tiny-icon" />
        <span className="repo-select-text">{selectLabel}</span>
      </button>
    </div>
  );
}
