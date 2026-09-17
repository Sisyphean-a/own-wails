import {
    ApproveHistoryDeletePlan,
    BuildDeletePlan,
    ListHistoryThreads,
    RunReadOnlyScan,
} from '../wailsjs/go/main/App';
import type {HistoryApproveResult, HistoryExecutionResult, HistoryPlanResult} from './history-types';
import type {WorkspaceState} from './workspace-types';

export async function fetchReadOnlyWorkspace(): Promise<WorkspaceState> {
    const scan = await RunReadOnlyScan();
    const plan = await BuildDeletePlan(scan.manifestPath);
    return {kind: 'ready', scan, plan};
}

export async function fetchHistoryThreads() {
    return ListHistoryThreads({limit: -1, grep: '', cwd: '', archived: false, all: true});
}

export async function approvePlan(planResult: HistoryPlanResult): Promise<HistoryApproveResult> {
    return ApproveHistoryDeletePlan({planPath: planResult.planPath});
}

export async function runWithLoading<T>(
    kind: T,
    setLoading: (value: T | null) => void,
    setError: (value: string | null) => void,
    task: () => Promise<void>,
) {
    setLoading(kind);
    setError(null);
    try {
        await task();
    } catch (cause) {
        setError(String(cause));
    } finally {
        setLoading(null);
    }
}

export function buildEvidenceRequest(args: {
    planResult: HistoryPlanResult | null;
    executionResult: HistoryExecutionResult | null;
}) {
    return {
        runId: args.executionResult?.runId ?? args.planResult?.runId ?? '',
        discoveryPath: '',
        manifestBeforePath: '',
        duplicateGroupsPath: '',
        deletePlanPath: args.planResult?.planPath ?? '',
        approvedPlanPath: args.executionResult?.approvedPlanPath ?? '',
        rollbackJournalPath: args.executionResult?.rollbackJournalPath ?? '',
        execResultPath: args.executionResult?.execResultPath ?? '',
        manifestAfterPath: args.executionResult?.manifestAfterPath ?? '',
        architecturePath: '',
        contextPath: '',
        historyPath: '',
    };
}
