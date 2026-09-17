export type ScanResult = {
    runId: string;
    roots: string[];
    discoveryPath: string;
    manifestPath: string;
    unknownItemsPath: string;
    summary: { rootCount: number; itemCount: number; unknownCount: number };
    items: Array<{
        sourceRoot: string;
        path: string;
        kind: string;
        size: number;
        mtimeUtc: string;
        attributes: string[];
        linkType?: string;
        target?: string;
    }>;
};

export type DeletePlanItem = {
    duplicateGroup: string;
    sessionUid?: string;
    sourcePath: string;
    preferredPath: string;
    action: string;
    reasonCode: string;
    reason: string;
    requiresCli: boolean;
    reviewNeeded: boolean;
    quarantinePath?: string;
    warnings: string[];
};

export type GroupCandidate = {
    sessionUid?: string;
    threadUid?: string;
    storageKind: string;
    sourcePath: string;
    canonicalPath: string;
    realPath: string;
    updatedAt: string;
    preferred: boolean;
    relation: string;
    action: string;
    reasonCode: string;
    reason: string;
    requiresCli: boolean;
    reviewNeeded: boolean;
    quarantinePath?: string;
    warnings: string[];
};

export type DuplicateGroup = {
    duplicateGroup: string;
    preferredPath: string;
    reviewNeeded: boolean;
    warning: string;
    candidates: GroupCandidate[];
};

export type DeletePlanResult = {
    runId: string;
    manifestPath: string;
    duplicateGroupsPath: string;
    deletePlanPath: string;
    summary: {
        groupCount: number;
        candidateCount: number;
        reviewCount: number;
        plannedCount: number;
    };
    groups: DuplicateGroup[];
    items: DeletePlanItem[];
    warnings: string[];
};

export type WorkspaceState =
    | { kind: 'idle' }
    | { kind: 'running' }
    | { kind: 'ready'; scan: ScanResult; plan: DeletePlanResult }
    | { kind: 'error'; message: string };
