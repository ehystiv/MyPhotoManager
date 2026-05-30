export namespace main {
	
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
	    }
	}
	export class ScanResult {
	    total: number;
	    raw: number;
	    others: number;
	    noExif: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.raw = source["raw"];
	        this.others = source["others"];
	        this.noExif = source["noExif"];
	    }
	}

}

