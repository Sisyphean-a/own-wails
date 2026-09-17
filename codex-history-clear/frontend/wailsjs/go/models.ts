export namespace history {
	
	export class ApproveRequest {
	    planPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ApproveRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.planPath = source["planPath"];
	    }
	}
	export class PlanStore {
	    store: string;
	    path: string;
	    action: string;
	    detail: string;
	    count: number;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlanStore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.store = source["store"];
	        this.path = source["path"];
	        this.action = source["action"];
	        this.detail = source["detail"];
	        this.count = source["count"];
	        this.exists = source["exists"];
	    }
	}
	export class ThreadSummary {
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
	
	    static createFrom(source: any = {}) {
	        return new ThreadSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.sourceTitle = source["sourceTitle"];
	        this.source = source["source"];
	        this.modelProvider = source["modelProvider"];
	        this.threadSource = source["threadSource"];
	        this.rolloutPath = source["rolloutPath"];
	        this.rolloutPaths = source["rolloutPaths"];
	        this.isClone = source["isClone"];
	        this.clonedFrom = source["clonedFrom"];
	        this.originalProvider = source["originalProvider"];
	        this.registered = source["registered"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.cwd = source["cwd"];
	        this.archived = source["archived"];
	        this.sizeBytes = source["sizeBytes"];
	        this.firstUserMessage = source["firstUserMessage"];
	        this.preview = source["preview"];
	    }
	}
	export class PlanTarget {
	    thread: ThreadSummary;
	    stores: PlanStore[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new PlanTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.thread = this.convertValues(source["thread"], ThreadSummary);
	        this.stores = this.convertValues(source["stores"], PlanStore);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlanSummary {
	    targetCount: number;
	    storeCount: number;
	    warningCount: number;
	
	    static createFrom(source: any = {}) {
	        return new PlanSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetCount = source["targetCount"];
	        this.storeCount = source["storeCount"];
	        this.warningCount = source["warningCount"];
	    }
	}
	export class ApproveResult {
	    runId: string;
	    planPath: string;
	    approvedPlanPath: string;
	    summary: PlanSummary;
	    targets: PlanTarget[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ApproveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.planPath = source["planPath"];
	        this.approvedPlanPath = source["approvedPlanPath"];
	        this.summary = this.convertValues(source["summary"], PlanSummary);
	        this.targets = this.convertValues(source["targets"], PlanTarget);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupArtifact {
	    originalPath: string;
	    backupPath: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupArtifact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.backupPath = source["backupPath"];
	    }
	}
	export class BuildPlanRequest {
	    threadIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new BuildPlanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threadIds = source["threadIds"];
	    }
	}
	export class EvidencePackArtifact {
	    label: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidencePackArtifact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.path = source["path"];
	    }
	}
	export class EvidencePackRequest {
	    runId: string;
	    discoveryPath: string;
	    manifestBeforePath: string;
	    duplicateGroupsPath: string;
	    deletePlanPath: string;
	    approvedPlanPath: string;
	    rollbackJournalPath: string;
	    execResultPath: string;
	    manifestAfterPath: string;
	    architecturePath: string;
	    contextPath: string;
	    historyPath: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidencePackRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.discoveryPath = source["discoveryPath"];
	        this.manifestBeforePath = source["manifestBeforePath"];
	        this.duplicateGroupsPath = source["duplicateGroupsPath"];
	        this.deletePlanPath = source["deletePlanPath"];
	        this.approvedPlanPath = source["approvedPlanPath"];
	        this.rollbackJournalPath = source["rollbackJournalPath"];
	        this.execResultPath = source["execResultPath"];
	        this.manifestAfterPath = source["manifestAfterPath"];
	        this.architecturePath = source["architecturePath"];
	        this.contextPath = source["contextPath"];
	        this.historyPath = source["historyPath"];
	    }
	}
	export class EvidencePackResult {
	    runId: string;
	    evidencePackPath: string;
	    artifacts: EvidencePackArtifact[];
	
	    static createFrom(source: any = {}) {
	        return new EvidencePackResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.evidencePackPath = source["evidencePackPath"];
	        this.artifacts = this.convertValues(source["artifacts"], EvidencePackArtifact);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExecuteRequest {
	    planPath: string;
	    confirmed: boolean;
	    backupOnly: boolean;
	    skipBackup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.planPath = source["planPath"];
	        this.confirmed = source["confirmed"];
	        this.backupOnly = source["backupOnly"];
	        this.skipBackup = source["skipBackup"];
	    }
	}
	export class VerificationFinding {
	    store: string;
	    path: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new VerificationFinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.store = source["store"];
	        this.path = source["path"];
	        this.detail = source["detail"];
	    }
	}
	export class VerificationResult {
	    status: string;
	    summary: string;
	    success: boolean;
	    remainingReferences: VerificationFinding[];
	
	    static createFrom(source: any = {}) {
	        return new VerificationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.summary = source["summary"];
	        this.success = source["success"];
	        this.remainingReferences = this.convertValues(source["remainingReferences"], VerificationFinding);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobEvent {
	    phase: string;
	    itemIndex: number;
	    itemTotal: number;
	    level: string;
	    message: string;
	    artifactPath: string;
	
	    static createFrom(source: any = {}) {
	        return new JobEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.itemIndex = source["itemIndex"];
	        this.itemTotal = source["itemTotal"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.artifactPath = source["artifactPath"];
	    }
	}
	export class MutationResult {
	    store: string;
	    action: string;
	    path: string;
	    changedRows: number;
	    changed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MutationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.store = source["store"];
	        this.action = source["action"];
	        this.path = source["path"];
	        this.changedRows = source["changedRows"];
	        this.changed = source["changed"];
	    }
	}
	export class ExecuteResult {
	    runId: string;
	    mode: string;
	    planPath: string;
	    approvedPlanPath: string;
	    rollbackJournalPath: string;
	    execResultPath: string;
	    manifestAfterPath: string;
	    backups: BackupArtifact[];
	    mutations: MutationResult[];
	    events: JobEvent[];
	    verification: VerificationResult;
	
	    static createFrom(source: any = {}) {
	        return new ExecuteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.mode = source["mode"];
	        this.planPath = source["planPath"];
	        this.approvedPlanPath = source["approvedPlanPath"];
	        this.rollbackJournalPath = source["rollbackJournalPath"];
	        this.execResultPath = source["execResultPath"];
	        this.manifestAfterPath = source["manifestAfterPath"];
	        this.backups = this.convertValues(source["backups"], BackupArtifact);
	        this.mutations = this.convertValues(source["mutations"], MutationResult);
	        this.events = this.convertValues(source["events"], JobEvent);
	        this.verification = this.convertValues(source["verification"], VerificationResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ListRequest {
	    limit: number;
	    cwd: string;
	    grep: string;
	    archived: boolean;
	    all: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limit = source["limit"];
	        this.cwd = source["cwd"];
	        this.grep = source["grep"];
	        this.archived = source["archived"];
	        this.all = source["all"];
	    }
	}
	export class ScanWarning {
	    path: string;
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanWarning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class ListSummary {
	    count: number;
	    limit: number;
	    hasMore: boolean;
	    warningCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ListSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.limit = source["limit"];
	        this.hasMore = source["hasMore"];
	        this.warningCount = source["warningCount"];
	    }
	}
	export class ListResult {
	    codexHome: string;
	    summary: ListSummary;
	    items: ThreadSummary[];
	    warnings: ScanWarning[];
	
	    static createFrom(source: any = {}) {
	        return new ListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codexHome = source["codexHome"];
	        this.summary = this.convertValues(source["summary"], ListSummary);
	        this.items = this.convertValues(source["items"], ThreadSummary);
	        this.warnings = this.convertValues(source["warnings"], ScanWarning);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PlanResult {
	    runId: string;
	    codexHome: string;
	    planPath: string;
	    summary: PlanSummary;
	    targets: PlanTarget[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new PlanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.codexHome = source["codexHome"];
	        this.planPath = source["planPath"];
	        this.summary = this.convertValues(source["summary"], PlanSummary);
	        this.targets = this.convertValues(source["targets"], PlanTarget);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class RollbackEntry {
	    originalPath: string;
	    backupPath: string;
	    restored: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RollbackEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.backupPath = source["backupPath"];
	        this.restored = source["restored"];
	    }
	}
	export class RollbackRequest {
	    journalPath: string;
	
	    static createFrom(source: any = {}) {
	        return new RollbackRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.journalPath = source["journalPath"];
	    }
	}
	export class RollbackResult {
	    runId: string;
	    journalPath: string;
	    restoredCount: number;
	    entries: RollbackEntry[];
	    events: JobEvent[];
	
	    static createFrom(source: any = {}) {
	        return new RollbackResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.journalPath = source["journalPath"];
	        this.restoredCount = source["restoredCount"];
	        this.entries = this.convertValues(source["entries"], RollbackEntry);
	        this.events = this.convertValues(source["events"], JobEvent);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	

}

export namespace main {
	
	export class CleanupWorkspaceConfig {
	    codexHome: string;
	    backupRoot: string;
	    usingDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CleanupWorkspaceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codexHome = source["codexHome"];
	        this.backupRoot = source["backupRoot"];
	        this.usingDefault = source["usingDefault"];
	    }
	}
	export class DeletePlanItem {
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
	
	    static createFrom(source: any = {}) {
	        return new DeletePlanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duplicateGroup = source["duplicateGroup"];
	        this.sessionUid = source["sessionUid"];
	        this.sourcePath = source["sourcePath"];
	        this.preferredPath = source["preferredPath"];
	        this.action = source["action"];
	        this.reasonCode = source["reasonCode"];
	        this.reason = source["reason"];
	        this.requiresCli = source["requiresCli"];
	        this.reviewNeeded = source["reviewNeeded"];
	        this.quarantinePath = source["quarantinePath"];
	        this.warnings = source["warnings"];
	    }
	}
	export class GroupCandidate {
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
	
	    static createFrom(source: any = {}) {
	        return new GroupCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionUid = source["sessionUid"];
	        this.threadUid = source["threadUid"];
	        this.storageKind = source["storageKind"];
	        this.sourcePath = source["sourcePath"];
	        this.canonicalPath = source["canonicalPath"];
	        this.realPath = source["realPath"];
	        this.updatedAt = source["updatedAt"];
	        this.preferred = source["preferred"];
	        this.relation = source["relation"];
	        this.action = source["action"];
	        this.reasonCode = source["reasonCode"];
	        this.reason = source["reason"];
	        this.requiresCli = source["requiresCli"];
	        this.reviewNeeded = source["reviewNeeded"];
	        this.quarantinePath = source["quarantinePath"];
	        this.warnings = source["warnings"];
	    }
	}
	export class DuplicateGroup {
	    duplicateGroup: string;
	    preferredPath: string;
	    reviewNeeded: boolean;
	    warning: string;
	    candidates: GroupCandidate[];
	
	    static createFrom(source: any = {}) {
	        return new DuplicateGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duplicateGroup = source["duplicateGroup"];
	        this.preferredPath = source["preferredPath"];
	        this.reviewNeeded = source["reviewNeeded"];
	        this.warning = source["warning"];
	        this.candidates = this.convertValues(source["candidates"], GroupCandidate);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlanSummary {
	    groupCount: number;
	    candidateCount: number;
	    reviewCount: number;
	    plannedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new PlanSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groupCount = source["groupCount"];
	        this.candidateCount = source["candidateCount"];
	        this.reviewCount = source["reviewCount"];
	        this.plannedCount = source["plannedCount"];
	    }
	}
	export class DeletePlanResult {
	    runId: string;
	    manifestPath: string;
	    duplicateGroupsPath: string;
	    deletePlanPath: string;
	    summary: PlanSummary;
	    groups: DuplicateGroup[];
	    items: DeletePlanItem[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new DeletePlanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.manifestPath = source["manifestPath"];
	        this.duplicateGroupsPath = source["duplicateGroupsPath"];
	        this.deletePlanPath = source["deletePlanPath"];
	        this.summary = this.convertValues(source["summary"], PlanSummary);
	        this.groups = this.convertValues(source["groups"], DuplicateGroup);
	        this.items = this.convertValues(source["items"], DeletePlanItem);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiscoveryItem {
	    sourceRoot: string;
	    path: string;
	    kind: string;
	    size: number;
	    mtimeUtc: string;
	    attributes: string[];
	    linkType?: string;
	    target?: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRoot = source["sourceRoot"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.mtimeUtc = source["mtimeUtc"];
	        this.attributes = source["attributes"];
	        this.linkType = source["linkType"];
	        this.target = source["target"];
	    }
	}
	
	
	
	export class ScanSummary {
	    rootCount: number;
	    itemCount: number;
	    unknownCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootCount = source["rootCount"];
	        this.itemCount = source["itemCount"];
	        this.unknownCount = source["unknownCount"];
	    }
	}
	export class ScanResult {
	    runId: string;
	    roots: string[];
	    discoveryPath: string;
	    manifestPath: string;
	    unknownItemsPath: string;
	    summary: ScanSummary;
	    items: DiscoveryItem[];
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.roots = source["roots"];
	        this.discoveryPath = source["discoveryPath"];
	        this.manifestPath = source["manifestPath"];
	        this.unknownItemsPath = source["unknownItemsPath"];
	        this.summary = this.convertValues(source["summary"], ScanSummary);
	        this.items = this.convertValues(source["items"], DiscoveryItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

