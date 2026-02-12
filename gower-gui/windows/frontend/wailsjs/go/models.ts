export namespace main {
	
	export class ConfigBehavior {
	    change_interval: number;
	    auto_download: boolean;
	    respect_dark_mode: boolean;
	    multi_monitor: string;
	    from_favorites: boolean;
	    daemon_enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigBehavior(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.change_interval = source["change_interval"];
	        this.auto_download = source["auto_download"];
	        this.respect_dark_mode = source["respect_dark_mode"];
	        this.multi_monitor = source["multi_monitor"];
	        this.from_favorites = source["from_favorites"];
	        this.daemon_enabled = source["daemon_enabled"];
	    }
	}
	export class ConfigPaths {
	    wallpapers: string;
	    use_system_dir: boolean;
	    index_wallpapers: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPaths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wallpapers = source["wallpapers"];
	        this.use_system_dir = source["use_system_dir"];
	        this.index_wallpapers = source["index_wallpapers"];
	    }
	}
	export class ConfigPower {
	    pause_on_low_battery: boolean;
	    low_battery_threshold: number;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPower(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pause_on_low_battery = source["pause_on_low_battery"];
	        this.low_battery_threshold = source["low_battery_threshold"];
	    }
	}
	export class Provider {
	    key: string;
	    name: string;
	    enabled: boolean;
	    api_key?: string;
	    hasApiKey: boolean;
	    search_url?: string;
	    isCustom?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.api_key = source["api_key"];
	        this.hasApiKey = source["hasApiKey"];
	        this.search_url = source["search_url"];
	        this.isCustom = source["isCustom"];
	    }
	}
	export class GowerConfig {
	    paths: ConfigPaths;
	    behavior: ConfigBehavior;
	    power: ConfigPower;
	    providers: Record<string, Provider>;
	    generic_providers: Record<string, Provider>;
	
	    static createFrom(source: any = {}) {
	        return new GowerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = this.convertValues(source["paths"], ConfigPaths);
	        this.behavior = this.convertValues(source["behavior"], ConfigBehavior);
	        this.power = this.convertValues(source["power"], ConfigPower);
	        this.providers = this.convertValues(source["providers"], Provider, true);
	        this.generic_providers = this.convertValues(source["generic_providers"], Provider, true);
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
	export class Monitor {
	    Name: string;
	    Primary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Monitor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Primary = source["Primary"];
	    }
	}
	
	export class Wallpaper {
	    id: string;
	    ext: string;
	    thumbnail: string;
	    permalink: string;
	    post_url: string;
	    url: string;
	    link: string;
	    seen: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Wallpaper(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ext = source["ext"];
	        this.thumbnail = source["thumbnail"];
	        this.permalink = source["permalink"];
	        this.post_url = source["post_url"];
	        this.url = source["url"];
	        this.link = source["link"];
	        this.seen = source["seen"];
	    }
	}

}

