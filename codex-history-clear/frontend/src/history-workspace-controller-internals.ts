import type {Dispatch, SetStateAction} from 'react';
import {useEffect, useState} from 'react';
import {
    BuildHistoryDeletePlan,
    ExecuteHistoryDeletePlan,
    ExportHistoryEvidencePack,
    GetCleanupWorkspaceConfig,
    RollbackHistoryDelete,
    SetCleanupWorkspaceRoot,
} from '../wailsjs/go/main/App';
import {BrowserOpenURL} from '../wailsjs/runtime/runtime';
import {approvePlan, buildEvidenceRequest, fetchHistoryThreads, fetchReadOnlyWorkspace, runWithLoading} from './history-workspace-api';
import type {HistoryLoadingState} from './history-workspace-contract';
import {
    analyzeThreadDuplicates,
    type DuplicateAnalysis,
} from './history-workspace-duplicates';
import {
    backupDirectoryForRun,
    createFileUrl,
    filterThreads,
    projectOptions,
    selectedThreadIds,
    suggestedDeleteThreadIds,
    totalSelectedBytes,
} from './history-workspace-helpers';
import type {
    CleanupWorkspaceConfig,
    HistoryEvidencePackResult,
    HistoryExecutionResult,
    HistoryListResult,
    HistoryPlanResult,
    HistoryRollbackResult,
} from './history-types';
import type {WorkspaceState} from './workspace-types';
import {initialViewState, type ViewState} from './history-workspace-view-state';

function selectionResetSignature(view: ViewState) {
    return [
        view.archivedFilter,
        view.projectQuery.trim(),
        view.ageFilter,
        view.sizeFilter,
        view.diagnosisFilter,
        view.keepRecent ? 'keep' : 'drop',
        view.skipUnknown ? 'skip' : 'include',
    ].join('|');
}
export type ControllerStore = {
    view: ViewState;
    setView: Dispatch<SetStateAction<ViewState>>;
    loading: HistoryLoadingState;
    setLoading: (value: HistoryLoadingState) => void;
    error: string | null;
    setError: (value: string | null) => void;
    workspaceConfig: CleanupWorkspaceConfig | null;
    setWorkspaceConfig: (value: CleanupWorkspaceConfig | null) => void;
    scanWorkspace: WorkspaceState;
    setScanWorkspace: (value: WorkspaceState) => void;
    listResult: HistoryListResult | null;
    setListResult: (value: HistoryListResult | null) => void;
    planResult: HistoryPlanResult | null;
    setPlanResult: (value: HistoryPlanResult | null) => void;
    executionResult: HistoryExecutionResult | null;
    setExecutionResult: (value: HistoryExecutionResult | null) => void;
    rollbackResult: HistoryRollbackResult | null;
    setRollbackResult: (value: HistoryRollbackResult | null) => void;
    evidencePackResult: HistoryEvidencePackResult | null;
    setEvidencePackResult: (value: HistoryEvidencePackResult | null) => void;
    planSelectionSignature: string;
    setPlanSelectionSignature: (value: string) => void;
};
export type DerivedWorkspaceData = {
    allThreads: HistoryListResult['items'];
    visibleThreads: HistoryListResult['items'];
    duplicateAnalysis: DuplicateAnalysis;
    projectChoices: string[];
    visibleSuggestedIds: string[];
    selectedIds: string[];
    selectedSize: number;
    selectionSignature: string;
    unknownCount: number;
};
export function useControllerStore(): ControllerStore {
    const [view, setView] = useState(initialViewState);
    const [loading, setLoading] = useState<HistoryLoadingState>(null);
    const [error, setError] = useState<string | null>(null);
    const [workspaceConfig, setWorkspaceConfig] = useState<CleanupWorkspaceConfig | null>(null);
    const [scanWorkspace, setScanWorkspace] = useState<WorkspaceState>({kind: 'idle'});
    const [listResult, setListResult] = useState<HistoryListResult | null>(null);
    const [planResult, setPlanResult] = useState<HistoryPlanResult | null>(null);
    const [executionResult, setExecutionResult] = useState<HistoryExecutionResult | null>(null);
    const [rollbackResult, setRollbackResult] = useState<HistoryRollbackResult | null>(null);
    const [evidencePackResult, setEvidencePackResult] = useState<HistoryEvidencePackResult | null>(null);
    const [planSelectionSignature, setPlanSelectionSignature] = useState('');
    return {
        view, setView, loading, setLoading, error, setError, workspaceConfig, setWorkspaceConfig, scanWorkspace, setScanWorkspace,
        listResult, setListResult, planResult, setPlanResult, executionResult, setExecutionResult, rollbackResult, setRollbackResult,
        evidencePackResult, setEvidencePackResult, planSelectionSignature, setPlanSelectionSignature,
    };
}
export function useDerivedWorkspaceData(listResult: HistoryListResult | null, scanWorkspace: WorkspaceState, view: ViewState): DerivedWorkspaceData {
    const allThreads = listResult?.items ?? [];
    const duplicateAnalysis = analyzeThreadDuplicates(allThreads);
    const visibleThreads = filterThreads(allThreads, view, duplicateAnalysis);
    const visibleSuggestedIds = suggestedDeleteThreadIds(visibleThreads, duplicateAnalysis);
    const selectedIds = selectedThreadIds(visibleThreads, view);
    return {
        allThreads,
        visibleThreads,
        duplicateAnalysis,
        projectChoices: projectOptions(allThreads),
        visibleSuggestedIds,
        selectedIds,
        selectedSize: totalSelectedBytes(visibleThreads, selectedIds),
        selectionSignature: [...selectedIds].sort().join('|'),
        unknownCount: scanWorkspace.kind === 'ready' ? scanWorkspace.scan.summary.unknownCount : 0,
    };
}
export function useViewValueSetter(setView: ControllerStore['setView']) {
    return function setViewValue<K extends keyof ViewState>(key: K, value: ViewState[K]) {
        setView((current) => ({...current, [key]: value}));
    };
}
export function createPlanReset(store: ControllerStore) {
    return function resetPlanArtifacts() {
        store.setPlanResult(null);
        store.setExecutionResult(null);
        store.setRollbackResult(null);
        store.setEvidencePackResult(null);
        store.setPlanSelectionSignature('');
        store.setView((current) => ({...current, confirmText: ''}));
    };
}
export function useControllerEffects(args: {
    startScan: () => Promise<void>;
    allThreads: HistoryListResult['items'];
    projectChoices: string[];
    listResult: HistoryListResult | null;
    planSelectionSignature: string;
    selectionSignature: string;
    selectionResetKey: string;
    resetPlanArtifacts: () => void;
    setView: ControllerStore['setView'];
}) {
    useEffect(() => { void args.startScan(); }, []);
    useEffect(() => {
        const validIds = new Set(args.allThreads.map((thread) => thread.id));
        args.setView((current) => ({
            ...current,
            manualSelectedIds: current.manualSelectedIds.filter((id) => validIds.has(id)),
            selectedProject: current.selectedProject !== '' && !args.projectChoices.includes(current.selectedProject)
                ? args.projectChoices[0] ?? ''
                : current.selectedProject,
        }));
    }, [args.listResult]);
    useEffect(() => {
        if (args.planSelectionSignature !== '' && args.selectionSignature !== args.planSelectionSignature) {
            args.resetPlanArtifacts();
        }
    }, [args.selectionSignature, args.planSelectionSignature]);
}

export function useBoundControllerEffects(args: {
    store: Pick<ControllerStore, 'listResult' | 'planSelectionSignature' | 'setView'>;
    derived: Pick<DerivedWorkspaceData, 'allThreads' | 'projectChoices' | 'selectionSignature'>;
    view: ViewState;
    startScan: () => Promise<void>;
    resetPlanArtifacts: () => void;
}) {
    useControllerEffects({
        startScan: args.startScan,
        allThreads: args.derived.allThreads,
        projectChoices: args.derived.projectChoices,
        listResult: args.store.listResult,
        planSelectionSignature: args.store.planSelectionSignature,
        selectionSignature: args.derived.selectionSignature,
        selectionResetKey: selectionResetSignature(args.view),
        resetPlanArtifacts: args.resetPlanArtifacts,
        setView: args.store.setView,
    });
}

export function useWorkspaceActions(args: { store: ControllerStore; resetPlanArtifacts: () => void; }) {
    const refreshWorkspace = async (config: CleanupWorkspaceConfig | null = null) => {
        const [nextConfig, nextScanWorkspace, nextListResult] = await Promise.all([
            config ? Promise.resolve(config) : GetCleanupWorkspaceConfig(),
            fetchReadOnlyWorkspace(),
            fetchHistoryThreads(),
        ]);
        args.store.setWorkspaceConfig(nextConfig);
        args.store.setScanWorkspace(nextScanWorkspace);
        args.store.setListResult(nextListResult);
    };
    const startScan = () => runWithLoading('scan', args.store.setLoading, args.store.setError, async () => refreshWorkspace());
    const changeDirectory = async () => {
        const input = window.prompt('输入新的 .codex 目录。留空可恢复默认目录。', args.store.workspaceConfig?.codexHome ?? '');
        if (input === null) return;
        return runWithLoading('directory', args.store.setLoading, args.store.setError, async () => {
            const nextConfig = await SetCleanupWorkspaceRoot(input.trim());
            args.store.setView((current) => ({
                ...initialViewState(),
                strategy: current.strategy,
                autoBackup: current.autoBackup,
                keepRecent: current.keepRecent,
                generateReport: current.generateReport,
                skipUnknown: current.skipUnknown,
            }));
            args.resetPlanArtifacts();
            await refreshWorkspace(nextConfig);
        });
    };
    const openBackupDirectory = () => {
        const runId = args.store.executionResult?.runId ?? args.store.planResult?.runId;
        const path = backupDirectoryForRun(args.store.workspaceConfig, runId);
        if (path !== '') BrowserOpenURL(createFileUrl(path));
    };
    return {refreshWorkspace, startScan, changeDirectory, openBackupDirectory};
}
export function usePlanActions(args: {
    store: ControllerStore;
    selectedIds: string[];
    selectionSignature: string;
    resetPlanArtifacts: () => void;
    refreshWorkspace: (config?: CleanupWorkspaceConfig | null) => Promise<void>;
}) {
    const buildPlan = () => args.selectedIds.length === 0 ? Promise.resolve() : runWithLoading('plan', args.store.setLoading, args.store.setError, async () => {
        const result = await BuildHistoryDeletePlan({threadIds: args.selectedIds});
        args.store.setPlanResult(result);
        args.store.setExecutionResult(null);
        args.store.setRollbackResult(null);
        args.store.setEvidencePackResult(null);
        args.store.setPlanSelectionSignature(args.selectionSignature);
        args.store.setView((current) => ({...current, confirmText: ''}));
    });
    const runExecution = (backupOnly: boolean) => !args.store.planResult ? Promise.resolve() : runWithLoading('execute', args.store.setLoading, args.store.setError, async () => {
        const approval = await approvePlan(args.store.planResult!);
        const result = await ExecuteHistoryDeletePlan({
            planPath: approval.approvedPlanPath,
            confirmed: true,
            backupOnly,
            skipBackup: !backupOnly && !args.store.view.autoBackup,
        });
        args.store.setExecutionResult(result);
        args.store.setRollbackResult(null);
        args.store.setEvidencePackResult(null);
        args.store.setView((current) => ({...current, confirmText: ''}));
        if (args.store.view.generateReport) args.store.setEvidencePackResult(await ExportHistoryEvidencePack(buildEvidenceRequest({planResult: args.store.planResult, executionResult: result})));
        if (!backupOnly) await args.refreshWorkspace(args.store.workspaceConfig);
    });
    const rollbackPlan = () => !args.store.executionResult || args.store.executionResult.rollbackJournalPath === '' ? Promise.resolve() : runWithLoading('rollback', args.store.setLoading, args.store.setError, async () => {
        args.store.setRollbackResult(await RollbackHistoryDelete({journalPath: args.store.executionResult!.rollbackJournalPath}));
        await args.refreshWorkspace(args.store.workspaceConfig);
    });
    const exportEvidencePack = () => !args.store.planResult && !args.store.executionResult ? Promise.resolve() : runWithLoading('export', args.store.setLoading, args.store.setError, async () => {
        args.store.setEvidencePackResult(await ExportHistoryEvidencePack(buildEvidenceRequest({planResult: args.store.planResult, executionResult: args.store.executionResult})));
    });
    return {buildPlan, backupPlan: () => runExecution(true), executePlan: () => runExecution(false), rollbackPlan, exportEvidencePack};
}
