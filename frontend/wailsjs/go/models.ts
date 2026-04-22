export namespace main {
	
	export class ServerStatus {
	    running: boolean;
	    model: string;
	    gpu: boolean;
	    version: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.model = source["model"];
	        this.gpu = source["gpu"];
	        this.version = source["version"];
	        this.url = source["url"];
	    }
	}

}

