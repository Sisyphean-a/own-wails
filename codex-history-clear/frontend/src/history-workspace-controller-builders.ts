import {backupDirectoryForRun, buildRiskNotes, formatBytes, latestUpdatedAt, selectedProjects} from './history-workspace-helpers';
import type {ControllerStore} from './history-workspace-controller-internals';
import type {HistoryListResult} from './history-types';
import type {ViewState} from './history-workspace-view-state';

type ViewValueSetter = <K extends keyof ViewState>(key: K, value: ViewState[K]) => void;

type WorkspaceActions = {
    startScan: () => Promise<void>;
    changeDirectory: () => Promise<void>;
    openBackupDirectory: () => void;
};

type SelectionActions = {
    selectSuggested: () => void;
    toggleAllVisible: () => void;
    toggleSelected: (threadID: string) => void;
};

type PlanActions = {
    buildPlan: () => Promise<void>;
    backupPlan: () => Promise<void>;
    executePlan: () => Promise<void>;
    rollbackPlan: () => Promise<void>;
    exportEvidencePack: () => Promise<void>;
};

export function buildOverview(args: {
    allThreads: HistoryListResult['items'];
    suggestedCount: number;
    selectedSize: number;
    executionResult: ControllerStore['executionResult'];
}) {
    const rollbackText = !args.executionResult
        ? '待清理后启用'
        : args.executionResult.rollbackJournalPath !== ''
            ? '可回滚'
            : '未保留恢复点';
    return {
        totalSessions: args.allThreads.length,
        archivedSessions: args.allThreads.filter((thread) => thread.archived).length,
        suggestedCount: args.suggestedCount,
        releaseText: formatBytes(args.selectedSize),
        latestUpdate: latestUpdatedAt(args.allThreads),
        rollbackText,
    };
}

export function buildFilters(view: ViewState, setViewValue: ViewValueSetter) {
    return {
        titleQuery: view.titleQuery,
        setTitleQuery: (value: string) => setViewValue('titleQuery', value),
        projectQuery: view.projectQuery,
        setProjectQuery: (value: string) => setViewValue('projectQuery', value),
        archivedFilter: view.archivedFilter,
        setArchivedFilter: (value: ViewState['archivedFilter']) => setViewValue('archivedFilter', value),
        ageFilter: view.ageFilter,
        setAgeFilter: (value: ViewState['ageFilter']) => setViewValue('ageFilter', value),
        sizeFilter: view.sizeFilter,
        setSizeFilter: (value: ViewState['sizeFilter']) => setViewValue('sizeFilter', value),
        diagnosisFilter: view.diagnosisFilter,
        setDiagnosisFilter: (value: ViewState['diagnosisFilter']) => setViewValue('diagnosisFilter', value),
    };
}

export function buildStrategyState(
    view: ViewState,
    setViewValue: ViewValueSetter,
    chooseStrategy: (value: ViewState['strategy']) => void,
) {
    return {
        strategy: view.strategy,
        chooseStrategy,
        selectedProject: view.selectedProject,
        setSelectedProject: (value: string) => setViewValue('selectedProject', value),
        autoBackup: view.autoBackup,
        setAutoBackup: (value: boolean) => setViewValue('autoBackup', value),
        keepRecent: view.keepRecent,
        setKeepRecent: (value: boolean) => setViewValue('keepRecent', value),
        generateReport: view.generateReport,
        setGenerateReport: (value: boolean) => setViewValue('generateReport', value),
        skipUnknown: view.skipUnknown,
        setSkipUnknown: (value: boolean) => setViewValue('skipUnknown', value),
    };
}

export function buildPlanState(args: {
    store: ControllerStore;
    view: ViewState;
    visibleThreads: HistoryListResult['items'];
    selectedIds: string[];
    unknownCount: number;
    setViewValue: ViewValueSetter;
}) {
    return {
        planResult: args.store.planResult,
        executionResult: args.store.executionResult,
        rollbackResult: args.store.rollbackResult,
        evidencePackResult: args.store.evidencePackResult,
        confirmText: args.view.confirmText,
        setConfirmText: (value: string) => args.setViewValue('confirmText', value),
        canConfirm: args.view.confirmText.trim() === 'DELETE',
        backupPath: args.view.autoBackup || (args.store.executionResult?.backups.length ?? 0) > 0
            ? backupDirectoryForRun(args.store.workspaceConfig, args.store.executionResult?.runId ?? args.store.planResult?.runId)
            : '',
        affectedProjects: selectedProjects(args.visibleThreads, args.selectedIds),
        riskNotes: buildRiskNotes({
            autoBackup: args.view.autoBackup,
            selectedIds: args.selectedIds,
            threads: args.visibleThreads,
            planResult: args.store.planResult,
            unknownCount: args.unknownCount,
            keepRecent: args.view.keepRecent,
        }),
    };
}

export function buildActionSet(
    workspaceActions: WorkspaceActions,
    selectionActions: SelectionActions,
    planActions: PlanActions,
) {
    return {
        startScan: workspaceActions.startScan,
        changeDirectory: workspaceActions.changeDirectory,
        openBackupDirectory: workspaceActions.openBackupDirectory,
        ...selectionActions,
        ...planActions,
    };
}
