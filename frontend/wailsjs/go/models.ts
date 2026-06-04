export namespace main {
	
	export class CullingApplyResult {
	    deleted: number;
	    moved: number;
	    kept: number;
	    errors: number;
	    dryRun: boolean;
	    err?: string;
	
	    static createFrom(source: any = {}) {
	        return new CullingApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deleted = source["deleted"];
	        this.moved = source["moved"];
	        this.kept = source["kept"];
	        this.errors = source["errors"];
	        this.dryRun = source["dryRun"];
	        this.err = source["err"];
	    }
	}
	export class CullingPhoto {
	    path: string;
	    name: string;
	    rel: string;
	    mark: string;
	
	    static createFrom(source: any = {}) {
	        return new CullingPhoto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.rel = source["rel"];
	        this.mark = source["mark"];
	    }
	}
	export class CullingListResult {
	    root: string;
	    photos: CullingPhoto[];
	    err?: string;
	
	    static createFrom(source: any = {}) {
	        return new CullingListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.photos = this.convertValues(source["photos"], CullingPhoto);
	        this.err = source["err"];
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
	
	export class DestFolder {
	    path: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DestFolder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.count = source["count"];
	    }
	}
	export class DestCategory {
	    name: string;
	    count: number;
	    folders: DestFolder[];
	
	    static createFrom(source: any = {}) {
	        return new DestCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	        this.folders = this.convertValues(source["folders"], DestFolder);
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
	
	export class DestTreeResult {
	    outputDir: string;
	    categories: DestCategory[];
	    total: number;
	    scanned: number;
	    truncated: boolean;
	    err?: string;
	
	    static createFrom(source: any = {}) {
	        return new DestTreeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputDir = source["outputDir"];
	        this.categories = this.convertValues(source["categories"], DestCategory);
	        this.total = source["total"];
	        this.scanned = source["scanned"];
	        this.truncated = source["truncated"];
	        this.err = source["err"];
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
	export class FormatPreviewResult {
	    folder: string;
	    file: string;
	    full: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new FormatPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.file = source["file"];
	        this.full = source["full"];
	        this.error = source["error"];
	    }
	}
	export class HistoryEntry {
	    // Go type: time
	    runAt: any;
	    inputDir: string;
	    moved: number;
	    raw: number;
	    others: number;
	    skipped: number;
	    dupes: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runAt = this.convertValues(source["runAt"], null);
	        this.inputDir = source["inputDir"];
	        this.moved = source["moved"];
	        this.raw = source["raw"];
	        this.others = source["others"];
	        this.skipped = source["skipped"];
	        this.dupes = source["dupes"];
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
	export class PhotoMetaResult {
	    date?: string;
	    camera?: string;
	    lens?: string;
	    focal?: string;
	    aperture?: string;
	    shutter?: string;
	    iso?: string;
	    flash: boolean;
	    gps?: string;
	    width?: number;
	    height?: number;
	    bias?: string;
	    program?: string;
	    metering?: string;
	    maxAp?: string;
	
	    static createFrom(source: any = {}) {
	        return new PhotoMetaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.camera = source["camera"];
	        this.lens = source["lens"];
	        this.focal = source["focal"];
	        this.aperture = source["aperture"];
	        this.shutter = source["shutter"];
	        this.iso = source["iso"];
	        this.flash = source["flash"];
	        this.gps = source["gps"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.bias = source["bias"];
	        this.program = source["program"];
	        this.metering = source["metering"];
	        this.maxAp = source["maxAp"];
	    }
	}
	export class Prefs {
	    inputDir: string;
	    outputDir: string;
	    dryRun: boolean;
	    copyMode: boolean;
	    stripMeta: boolean;
	    modTime: boolean;
	    checkDupes: boolean;
	    renameOnly: boolean;
	    cleanDirs: boolean;
	    folderFmt: string;
	    fileTpl: string;
	    rawSplit: string;
	    recents: string[];
	    confirmedUnsafeOnce: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Prefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputDir = source["inputDir"];
	        this.outputDir = source["outputDir"];
	        this.dryRun = source["dryRun"];
	        this.copyMode = source["copyMode"];
	        this.stripMeta = source["stripMeta"];
	        this.modTime = source["modTime"];
	        this.checkDupes = source["checkDupes"];
	        this.renameOnly = source["renameOnly"];
	        this.cleanDirs = source["cleanDirs"];
	        this.folderFmt = source["folderFmt"];
	        this.fileTpl = source["fileTpl"];
	        this.rawSplit = source["rawSplit"];
	        this.recents = source["recents"];
	        this.confirmedUnsafeOnce = source["confirmedUnsafeOnce"];
	    }
	}
	export class ScanResult {
	    total: number;
	    raw: number;
	    others: number;
	    noExif: number;
	    totalBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.raw = source["raw"];
	        this.others = source["others"];
	        this.noExif = source["noExif"];
	        this.totalBytes = source["totalBytes"];
	    }
	}

}

