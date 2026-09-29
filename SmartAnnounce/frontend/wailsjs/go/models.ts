export namespace broadcast {
	
	export class ExportRequest {
	    sourcePath: string;
	    fileName: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourcePath = source["sourcePath"];
	        this.fileName = source["fileName"];
	    }
	}
	export class ExportResult {
	    filePath: string;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.cancelled = source["cancelled"];
	    }
	}
	export class GenerateRequest {
	    text: string;
	    voiceId: string;
	    speed: number;
	    volume: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.voiceId = source["voiceId"];
	        this.speed = source["speed"];
	        this.volume = source["volume"];
	    }
	}
	export class GenerateResult {
	    fileName: string;
	    filePath: string;
	    audioDataUri: string;
	    voiceId: string;
	    characterCount: number;
	    durationMillis: number;
	    audioSampleRate: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.filePath = source["filePath"];
	        this.audioDataUri = source["audioDataUri"];
	        this.voiceId = source["voiceId"];
	        this.characterCount = source["characterCount"];
	        this.durationMillis = source["durationMillis"];
	        this.audioSampleRate = source["audioSampleRate"];
	    }
	}
	export class SettingsInfo {
	    model: string;
	    hasApiKey: boolean;
	    proxyMode: string;
	    proxyUrl: string;
	    customVoiceId: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.hasApiKey = source["hasApiKey"];
	        this.proxyMode = source["proxyMode"];
	        this.proxyUrl = source["proxyUrl"];
	        this.customVoiceId = source["customVoiceId"];
	    }
	}
	export class SettingsUpdate {
	    apiKey: string;
	    model: string;
	    proxyMode: string;
	    proxyUrl: string;
	    customVoiceId: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.proxyMode = source["proxyMode"];
	        this.proxyUrl = source["proxyUrl"];
	        this.customVoiceId = source["customVoiceId"];
	    }
	}
	export class TextHistoryEntry {
	    id: string;
	    text: string;
	    day: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new TextHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.day = source["day"];
	        this.createdAt = source["createdAt"];
	    }
	}

}

