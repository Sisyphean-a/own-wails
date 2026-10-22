import type { BranchSummary, CommitPage, DashboardState, Direction, RepositorySlot, SourceMode } from "./types";

function backend() {
  return window.go.main.App;
}

export function loadState(): Promise<DashboardState> {
  return backend().LoadState();
}

export function refreshState(): Promise<DashboardState> {
  return backend().Refresh();
}

export function selectRepository(slot: RepositorySlot): Promise<DashboardState> {
  return backend().SelectRepository(slot);
}

export function selectRepositoryWorktree(slot: RepositorySlot, path: string): Promise<DashboardState> {
  return backend().SelectRepositoryWorktree(slot, path);
}

export function listSourceBranches(): Promise<BranchSummary[]> {
  return backend().ListSourceBranches();
}

export function listSourceCommits(revision: string, offset: number): Promise<CommitPage> {
  return backend().ListSourceCommits(revision, offset);
}

export function selectSource(mode: SourceMode, revision: string): Promise<DashboardState> {
  return backend().SelectSource(mode, revision);
}

export function swapRepositories(): Promise<DashboardState> {
  return backend().SwapRepositories();
}

export function setDirection(direction: Direction): Promise<DashboardState> {
  return backend().SetDirection(direction);
}

export function saveConfig(): Promise<DashboardState> {
  return backend().SaveConfig();
}

export function syncRepositories(): Promise<DashboardState> {
  return backend().SyncRepositories();
}

export function commitTarget(message: string): Promise<DashboardState> {
  return backend().CommitTarget(message);
}

export function generateCommitMessage(): Promise<string> {
  return backend().GenerateCommitMessage();
}

export function setAICommitAPIKey(apiKey: string): Promise<DashboardState> {
  return backend().SetAICommitAPIKey(apiKey);
}

export function pushTarget(): Promise<DashboardState> {
  return backend().PushTarget();
}
