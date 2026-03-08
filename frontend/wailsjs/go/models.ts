export namespace main {
	
	export class GitChange {
	    path: string;
	    originalPath: string;
	    indexStatus: string;
	    worktreeStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new GitChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.indexStatus = source["indexStatus"];
	        this.worktreeStatus = source["worktreeStatus"];
	    }
	}
	export class ChangeListResult {
	    success: boolean;
	    error: string;
	    changes: GitChange[];
	
	    static createFrom(source: any = {}) {
	        return new ChangeListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.changes = this.convertValues(source["changes"], GitChange);
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
	
	export class GitResult {
	    success: boolean;
	    output: string;
	    error: string;
	    exitCode: number;
	
	    static createFrom(source: any = {}) {
	        return new GitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error = source["error"];
	        this.exitCode = source["exitCode"];
	    }
	}
	export class RepositoryOperationResult {
	    success: boolean;
	    error: string;
	    repository: string;
	    repositories: string[];
	    current: string;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RepositoryOperationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.repository = source["repository"];
	        this.repositories = source["repositories"];
	        this.current = source["current"];
	        this.cancelled = source["cancelled"];
	    }
	}
	export class SmartSyncResult {
	    success: boolean;
	    output: string;
	    error: string;
	    exitCode: number;
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new SmartSyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error = source["error"];
	        this.exitCode = source["exitCode"];
	        this.action = source["action"];
	    }
	}

}

