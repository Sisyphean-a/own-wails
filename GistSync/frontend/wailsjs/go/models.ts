export namespace appsvc {
	
	export class QuickDownloadRequest {
	    profileId: string;
	    conflictPolicy: string;
	    overwriteItemIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new QuickDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.conflictPolicy = source["conflictPolicy"];
	        this.overwriteItemIds = source["overwriteItemIds"];
	    }
	}
	export class QuickOperationItem {
	    itemId: string;
	    targetPath: string;
	    status: string;
	    reason: string;
	    diffPreview: string;
	    diffStatus: string;
	    diffLines: syncflow.DiffLine[];
	    addedLines: number;
	    removedLines: number;
	
	    static createFrom(source: any = {}) {
	        return new QuickOperationItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.targetPath = source["targetPath"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	        this.diffPreview = source["diffPreview"];
	        this.diffStatus = source["diffStatus"];
	        this.diffLines = this.convertValues(source["diffLines"], syncflow.DiffLine);
	        this.addedLines = source["addedLines"];
	        this.removedLines = source["removedLines"];
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
	export class QuickOperationSummary {
	    uploaded: number;
	    applied: number;
	    skipped: number;
	    conflicts: number;
	    errors: number;
	
	    static createFrom(source: any = {}) {
	        return new QuickOperationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uploaded = source["uploaded"];
	        this.applied = source["applied"];
	        this.skipped = source["skipped"];
	        this.conflicts = source["conflicts"];
	        this.errors = source["errors"];
	    }
	}
	export class QuickOperationResult {
	    operationId: string;
	    action: string;
	    profileId: string;
	    snapshotId: string;
	    requiresConflictResolution: boolean;
	    summary: QuickOperationSummary;
	    conflicts: QuickOperationItem[];
	    items: QuickOperationItem[];
	
	    static createFrom(source: any = {}) {
	        return new QuickOperationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationId = source["operationId"];
	        this.action = source["action"];
	        this.profileId = source["profileId"];
	        this.snapshotId = source["snapshotId"];
	        this.requiresConflictResolution = source["requiresConflictResolution"];
	        this.summary = this.convertValues(source["summary"], QuickOperationSummary);
	        this.conflicts = this.convertValues(source["conflicts"], QuickOperationItem);
	        this.items = this.convertValues(source["items"], QuickOperationItem);
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
	
	export class QuickUploadRequest {
	    profileId: string;
	
	    static createFrom(source: any = {}) {
	        return new QuickUploadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	    }
	}

}

export namespace settings {
	
	export class ProfileItem {
	    id: string;
	    sourcePathTemplate: string;
	    relativePath: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProfileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourcePathTemplate = source["sourcePathTemplate"];
	        this.relativePath = source["relativePath"];
	        this.enabled = source["enabled"];
	    }
	}
	export class Profile {
	    id: string;
	    name: string;
	    restoreMode: string;
	    restoreRoot: string;
	    enabled: boolean;
	    items: ProfileItem[];
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.restoreMode = source["restoreMode"];
	        this.restoreRoot = source["restoreRoot"];
	        this.enabled = source["enabled"];
	        this.items = this.convertValues(source["items"], ProfileItem);
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
	export class Data {
	    token: string;
	    masterPassword: string;
	    activeProfileId: string;
	    profiles: Profile[];
	    cloudBootstrapDone?: boolean;
	    syncPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new Data(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.masterPassword = source["masterPassword"];
	        this.activeProfileId = source["activeProfileId"];
	        this.profiles = this.convertValues(source["profiles"], Profile);
	        this.cloudBootstrapDone = source["cloudBootstrapDone"];
	        this.syncPath = source["syncPath"];
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

export namespace syncflow {
	
	export class DiffLine {
	    kind: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new DiffLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.text = source["text"];
	    }
	}
	export class ApplyConflict {
	    itemId: string;
	    targetPath: string;
	    diffPreview: string;
	    diffStatus: string;
	    diffLines: DiffLine[];
	    addedLines: number;
	    removedLines: number;
	
	    static createFrom(source: any = {}) {
	        return new ApplyConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.targetPath = source["targetPath"];
	        this.diffPreview = source["diffPreview"];
	        this.diffStatus = source["diffStatus"];
	        this.diffLines = this.convertValues(source["diffLines"], DiffLine);
	        this.addedLines = source["addedLines"];
	        this.removedLines = source["removedLines"];
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
	export class ApplyItemResult {
	    itemId: string;
	    targetPath: string;
	    status: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplyItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.targetPath = source["targetPath"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	    }
	}
	export class ApplySnapshotRequest {
	    profileId: string;
	    snapshotId: string;
	    masterPassword: string;
	    restoreMode: string;
	    restoreRoot: string;
	    selectedItemIds: string[];
	    overwriteItemIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new ApplySnapshotRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.snapshotId = source["snapshotId"];
	        this.masterPassword = source["masterPassword"];
	        this.restoreMode = source["restoreMode"];
	        this.restoreRoot = source["restoreRoot"];
	        this.selectedItemIds = source["selectedItemIds"];
	        this.overwriteItemIds = source["overwriteItemIds"];
	    }
	}
	export class ApplySnapshotResult {
	    applied: number;
	    skipped: number;
	    items: ApplyItemResult[];
	
	    static createFrom(source: any = {}) {
	        return new ApplySnapshotResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.skipped = source["skipped"];
	        this.items = this.convertValues(source["items"], ApplyItemResult);
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
	
	export class MasterPasswordValidation {
	    valid: boolean;
	    hasSnapshot: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MasterPasswordValidation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.hasSnapshot = source["hasSnapshot"];
	    }
	}
	export class SnapshotMeta {
	    id: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class UploadProfileResult {
	    snapshotId: string;
	    uploaded: number;
	
	    static createFrom(source: any = {}) {
	        return new UploadProfileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.snapshotId = source["snapshotId"];
	        this.uploaded = source["uploaded"];
	    }
	}

}

