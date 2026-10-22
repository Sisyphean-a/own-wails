export namespace syncer {
	
	export class AIProposal {
	    action: string;
	    summary: string;
	    warnings: string[];
	    content: string;
	    localHash: string;
	    remoteHash: string;
	
	    static createFrom(source: any = {}) {
	        return new AIProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.summary = source["summary"];
	        this.warnings = source["warnings"];
	        this.content = source["content"];
	        this.localHash = source["localHash"];
	        this.remoteHash = source["remoteHash"];
	    }
	}
	export class AISettings {
	    configured: boolean;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new AISettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.model = source["model"];
	    }
	}
	export class Comparison {
	    path: string;
	    local: string;
	    remote: string;
	    localHash: string;
	    remoteHash: string;
	
	    static createFrom(source: any = {}) {
	        return new Comparison(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.local = source["local"];
	        this.remote = source["remote"];
	        this.localHash = source["localHash"];
	        this.remoteHash = source["remoteHash"];
	    }
	}
	export class Device {
	    id: string;
	    name: string;
	    address: string;
	    seen: number;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.seen = source["seen"];
	    }
	}
	export class File {
	    hash: string;
	    size: number;
	    modified: number;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	    }
	}
	export class Invitation {
	    id: string;
	    name: string;
	    peerId: string;
	    remoteRoot: string;
	    coordinator: string;
	
	    static createFrom(source: any = {}) {
	        return new Invitation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.peerId = source["peerId"];
	        this.remoteRoot = source["remoteRoot"];
	        this.coordinator = source["coordinator"];
	    }
	}
	export class Issue {
	    path: string;
	    kind: string;
	    message: string;
	    local: File;
	    remote: File;
	    copy?: string;
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.message = source["message"];
	        this.local = this.convertValues(source["local"], File);
	        this.remote = this.convertValues(source["remote"], File);
	        this.copy = source["copy"];
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
	export class Pairing {
	    id: string;
	    peerId: string;
	    name: string;
	    address: string;
	    code: string;
	    localApproved: boolean;
	    remoteApproved: boolean;
	    expires: number;
	
	    static createFrom(source: any = {}) {
	        return new Pairing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.peerId = source["peerId"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.code = source["code"];
	        this.localApproved = source["localApproved"];
	        this.remoteApproved = source["remoteApproved"];
	        this.expires = source["expires"];
	    }
	}
	export class Peer {
	    id: string;
	    name: string;
	    address: string;
	    certificate?: number[];
	    online: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Peer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.certificate = source["certificate"];
	        this.online = source["online"];
	    }
	}
	export class Preview {
	    addedLocal: number;
	    addedRemote: number;
	    changed: number;
	    decisions: number;
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.addedLocal = source["addedLocal"];
	        this.addedRemote = source["addedRemote"];
	        this.changed = source["changed"];
	        this.decisions = source["decisions"];
	    }
	}
	export class Settings {
	    name: string;
	    backupDir: string;
	    port: number;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.backupDir = source["backupDir"];
	        this.port = source["port"];
	        this.autoStart = source["autoStart"];
	    }
	}
	export class Status {
	    taskId: string;
	    state: string;
	    message: string;
	    lastSync: number;
	    done: number;
	    total: number;
	    issues: Issue[];
	    skipped: string[];
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.state = source["state"];
	        this.message = source["message"];
	        this.lastSync = source["lastSync"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.skipped = source["skipped"];
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
	export class TaskView {
	    id: string;
	    name: string;
	    peerId: string;
	    root: string;
	    remoteRoot: string;
	    coordinator: string;
	    paused: boolean;
	    started: boolean;
	    status: Status;
	    peerName: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.peerId = source["peerId"];
	        this.root = source["root"];
	        this.remoteRoot = source["remoteRoot"];
	        this.coordinator = source["coordinator"];
	        this.paused = source["paused"];
	        this.started = source["started"];
	        this.status = this.convertValues(source["status"], Status);
	        this.peerName = source["peerName"];
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
	export class Snapshot {
	    id: string;
	    settings: Settings;
	    addresses: string[];
	    devices: Device[];
	    peers: Peer[];
	    pairings: Pairing[];
	    invitations: Invitation[];
	    tasks: TaskView[];
	    networkError: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.addresses = source["addresses"];
	        this.devices = this.convertValues(source["devices"], Device);
	        this.peers = this.convertValues(source["peers"], Peer);
	        this.pairings = this.convertValues(source["pairings"], Pairing);
	        this.invitations = this.convertValues(source["invitations"], Invitation);
	        this.tasks = this.convertValues(source["tasks"], TaskView);
	        this.networkError = source["networkError"];
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
	
	
	export class Version {
	    id: string;
	    taskId: string;
	    path: string;
	    hash: string;
	    size: number;
	    time: number;
	    reason: string;
	    backupDir?: string;
	
	    static createFrom(source: any = {}) {
	        return new Version(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.taskId = source["taskId"];
	        this.path = source["path"];
	        this.hash = source["hash"];
	        this.size = source["size"];
	        this.time = source["time"];
	        this.reason = source["reason"];
	        this.backupDir = source["backupDir"];
	    }
	}

}

