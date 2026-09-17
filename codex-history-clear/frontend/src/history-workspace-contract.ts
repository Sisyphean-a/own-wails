import type {AgeFilter, ArchivedFilter, CleanupStrategy, SizeFilter} from './history-workspace-helpers';
import type {DiagnosisFilter, DuplicateAnalysis} from './history-workspace-duplicates';
import type {
    CleanupWorkspaceConfig,
    HistoryEvidencePackResult,
    HistoryExecutionResult,
    HistoryListResult,
    HistoryPlanResult,
    HistoryRollbackResult,
} from './history-types';
import type {WorkspaceState} from './workspace-types';

export type HistoryLoadingState = 'scan' | 'plan' | 'execute' | 'rollback' | 'export' | 'directory' | null;

export type HistoryWorkspaceController = {
    loading: HistoryLoadingState;
    error: string | null;
    workspaceConfig: CleanupWorkspaceConfig | null;
    scanWorkspace: WorkspaceState;
    listResult: HistoryListResult | null;
    visibleThreads: HistoryListResult['items'];
    duplicateAnalysis: DuplicateAnalysis;
    projectChoices: string[];
    selectedIds: string[];
    confirmPhrase: string;
    overview: {
        totalSessions: number;
        archivedSessions: number;
        suggestedCount: number;
        releaseText: string;
        latestUpdate: string;
        rollbackText: string;
    };
    filters: {
        titleQuery: string;
        setTitleQuery: (value: string) => void;
        projectQuery: string;
        setProjectQuery: (value: string) => void;
        archivedFilter: ArchivedFilter;
        setArchivedFilter: (value: ArchivedFilter) => void;
        ageFilter: AgeFilter;
        setAgeFilter: (value: AgeFilter) => void;
        sizeFilter: SizeFilter;
        setSizeFilter: (value: SizeFilter) => void;
        diagnosisFilter: DiagnosisFilter;
        setDiagnosisFilter: (value: DiagnosisFilter) => void;
    };
    strategyState: {
        strategy: CleanupStrategy;
        chooseStrategy: (value: CleanupStrategy) => void;
        selectedProject: string;
        setSelectedProject: (value: string) => void;
        autoBackup: boolean;
        setAutoBackup: (value: boolean) => void;
        keepRecent: boolean;
        setKeepRecent: (value: boolean) => void;
        generateReport: boolean;
        setGenerateReport: (value: boolean) => void;
        skipUnknown: boolean;
        setSkipUnknown: (value: boolean) => void;
    };
    planState: {
        planResult: HistoryPlanResult | null;
        executionResult: HistoryExecutionResult | null;
        rollbackResult: HistoryRollbackResult | null;
        evidencePackResult: HistoryEvidencePackResult | null;
        confirmText: string;
        setConfirmText: (value: string) => void;
        canConfirm: boolean;
        backupPath: string;
        affectedProjects: string[];
        riskNotes: string[];
    };
    actions: {
        startScan: () => Promise<void>;
        changeDirectory: () => Promise<void>;
        openBackupDirectory: () => void;
        selectSuggested: () => void;
        toggleAllVisible: () => void;
        toggleSelected: (threadID: string) => void;
        buildPlan: () => Promise<void>;
        backupPlan: () => Promise<void>;
        executePlan: () => Promise<void>;
        rollbackPlan: () => Promise<void>;
        exportEvidencePack: () => Promise<void>;
    };
};
