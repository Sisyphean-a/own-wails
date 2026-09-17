export type HistoryThread = {
    id: string;
    title: string;
    sourceTitle: string;
    source: string;
    modelProvider: string;
    threadSource: string;
    rolloutPath: string;
	rolloutPaths: string[];
	isClone: boolean;
	clonedFrom: string;
	originalProvider: string;
	registered: boolean;
    createdAt: string;
    updatedAt: string;
    cwd: string;
    archived: boolean;
    sizeBytes: number;
    firstUserMessage: string;
    preview: string;
};

export type CleanupWorkspaceConfig = {
    codexHome: string;
    backupRoot: string;
    usingDefault: boolean;
};

export type HistoryListResult = {
    codexHome: string;
	summary: { count: number; limit: number; hasMore: boolean; warningCount: number };
    items: HistoryThread[];
	warnings: Array<{path: string; code: string; message: string}>;
};

export type HistoryPlanStore = {
    store: string;
    path: string;
    action: string;
    detail: string;
    count: number;
    exists: boolean;
};

export type HistoryPlanTarget = {
    thread: HistoryThread;
    stores: HistoryPlanStore[];
    warnings: string[];
};

export type HistoryPlanResult = {
    runId: string;
    codexHome: string;
    planPath: string;
    summary: { targetCount: number; storeCount: number; warningCount: number };
    targets: HistoryPlanTarget[];
    warnings: string[];
};

export type HistoryApproveResult = {
    runId: string;
    planPath: string;
    approvedPlanPath: string;
    summary: { targetCount: number; storeCount: number; warningCount: number };
    targets: HistoryPlanTarget[];
    warnings: string[];
};

export type HistoryExecutionResult = {
    runId: string;
    mode: string;
    planPath: string;
    approvedPlanPath: string;
    rollbackJournalPath: string;
    execResultPath: string;
    manifestAfterPath: string;
    backups: Array<{ originalPath: string; backupPath: string }>;
    mutations: Array<{ store: string; action: string; path: string; changedRows: number; changed: boolean }>;
    events: Array<{ phase: string; itemIndex: number; itemTotal: number; level: string; message: string; artifactPath: string }>;
    verification: {
        status: string;
        summary: string;
        success: boolean;
        remainingReferences: Array<{ store: string; path: string; detail: string }>;
    };
};

export type HistoryRollbackResult = {
    runId: string;
    journalPath: string;
    restoredCount: number;
    entries: Array<{ originalPath: string; backupPath: string; restored: boolean }>;
    events: Array<{ phase: string; itemIndex: number; itemTotal: number; level: string; message: string; artifactPath: string }>;
};

export type HistoryEvidencePackResult = {
    runId: string;
    evidencePackPath: string;
    artifacts: Array<{ label: string; path: string }>;
};
