export namespace auth {
	
	export class DeviceCodeResult {
	    userCode: string;
	    deviceCode: string;
	    verificationUri: string;
	    expiresIn: number;
	    interval: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceCodeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userCode = source["userCode"];
	        this.deviceCode = source["deviceCode"];
	        this.verificationUri = source["verificationUri"];
	        this.expiresIn = source["expiresIn"];
	        this.interval = source["interval"];
	        this.message = source["message"];
	    }
	}

}

export namespace models {
	
	export class Account {
	    id: string;
	    username: string;
	    type: string;
	    uuid: string;
	    accessToken: string;
	    refreshToken?: string;
	    expiresAt?: number;
	    skinUrl: string;
	    capeUrl: string;
	    isActive: boolean;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.type = source["type"];
	        this.uuid = source["uuid"];
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.expiresAt = source["expiresAt"];
	        this.skinUrl = source["skinUrl"];
	        this.capeUrl = source["capeUrl"];
	        this.isActive = source["isActive"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class AppSettings {
	    launcherDir: string;
	    defaultMinRam: number;
	    defaultMaxRam: number;
	    defaultJavaPath?: string;
	    autoSetupDone?: boolean;
	    defaultJvmArgs: string;
	    autoCloseOnLaunch: boolean;
	    language: string;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.launcherDir = source["launcherDir"];
	        this.defaultMinRam = source["defaultMinRam"];
	        this.defaultMaxRam = source["defaultMaxRam"];
	        this.defaultJavaPath = source["defaultJavaPath"];
	        this.autoSetupDone = source["autoSetupDone"];
	        this.defaultJvmArgs = source["defaultJvmArgs"];
	        this.autoCloseOnLaunch = source["autoCloseOnLaunch"];
	        this.language = source["language"];
	        this.theme = source["theme"];
	    }
	}
	export class CommunityServerEntry {
	    id: string;
	    name: string;
	    manifestUrl: string;
	    tag: string;
	    isFeatured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CommunityServerEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.manifestUrl = source["manifestUrl"];
	        this.tag = source["tag"];
	        this.isFeatured = source["isFeatured"];
	    }
	}
	export class JavaRuntime {
	    path: string;
	    version: string;
	    majorVersion: number;
	    is64Bit: boolean;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JavaRuntime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.version = source["version"];
	        this.majorVersion = source["majorVersion"];
	        this.is64Bit = source["is64Bit"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class EnvironmentSetupStatus {
	    hasJava17: boolean;
	    hasJava21: boolean;
	    hasJava8: boolean;
	    java17Path: string;
	    java21Path: string;
	    java8Path: string;
	    selectedJava: string;
	    chillTownReady: boolean;
	    allRuntimes: JavaRuntime[];
	    totalRamGb: number;
	    recommendedRam: number;
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentSetupStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasJava17 = source["hasJava17"];
	        this.hasJava21 = source["hasJava21"];
	        this.hasJava8 = source["hasJava8"];
	        this.java17Path = source["java17Path"];
	        this.java21Path = source["java21Path"];
	        this.java8Path = source["java8Path"];
	        this.selectedJava = source["selectedJava"];
	        this.chillTownReady = source["chillTownReady"];
	        this.allRuntimes = this.convertValues(source["allRuntimes"], JavaRuntime);
	        this.totalRamGb = source["totalRamGb"];
	        this.recommendedRam = source["recommendedRam"];
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
	export class Instance {
	    id: string;
	    name: string;
	    gameVersion: string;
	    versionType: string;
	    loader: string;
	    loaderVersion: string;
	    icon: string;
	    minMemoryMb: number;
	    maxMemoryMb: number;
	    jvmArgs: string;
	    javaPath: string;
	    path: string;
	    serverSyncUrl?: string;
	    serverId?: string;
	    serverVersion?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    lastPlayed: any;
	    playTimeMinutes: number;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.gameVersion = source["gameVersion"];
	        this.versionType = source["versionType"];
	        this.loader = source["loader"];
	        this.loaderVersion = source["loaderVersion"];
	        this.icon = source["icon"];
	        this.minMemoryMb = source["minMemoryMb"];
	        this.maxMemoryMb = source["maxMemoryMb"];
	        this.jvmArgs = source["jvmArgs"];
	        this.javaPath = source["javaPath"];
	        this.path = source["path"];
	        this.serverSyncUrl = source["serverSyncUrl"];
	        this.serverId = source["serverId"];
	        this.serverVersion = source["serverVersion"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.lastPlayed = this.convertValues(source["lastPlayed"], null);
	        this.playTimeMinutes = source["playTimeMinutes"];
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
	export class InstanceFileItem {
	    name: string;
	    fileName: string;
	    sizeBytes: number;
	    modTime: string;
	    enabled: boolean;
	    fileType: string;
	
	    static createFrom(source: any = {}) {
	        return new InstanceFileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.fileName = source["fileName"];
	        this.sizeBytes = source["sizeBytes"];
	        this.modTime = source["modTime"];
	        this.enabled = source["enabled"];
	        this.fileType = source["fileType"];
	    }
	}
	
	export class LatestVersion {
	    release: string;
	    snapshot: string;
	
	    static createFrom(source: any = {}) {
	        return new LatestVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.release = source["release"];
	        this.snapshot = source["snapshot"];
	    }
	}
	export class LauncherUpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    title: string;
	    changelog: string;
	    downloadUrl: string;
	    releaseDate: string;
	    sha256?: string;
	
	    static createFrom(source: any = {}) {
	        return new LauncherUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.title = source["title"];
	        this.changelog = source["changelog"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseDate = source["releaseDate"];
	        this.sha256 = source["sha256"];
	    }
	}
	export class LoaderVersionResult {
	    loader: string;
	    gameVersion: string;
	    recommended: string;
	    versions: string[];
	
	    static createFrom(source: any = {}) {
	        return new LoaderVersionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loader = source["loader"];
	        this.gameVersion = source["gameVersion"];
	        this.recommended = source["recommended"];
	        this.versions = source["versions"];
	    }
	}
	export class ModpackArchive {
	    url?: string;
	    urls?: string[];
	    sha256?: string;
	    size?: number;
	
	    static createFrom(source: any = {}) {
	        return new ModpackArchive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.urls = source["urls"];
	        this.sha256 = source["sha256"];
	        this.size = source["size"];
	    }
	}
	export class ModpackFile {
	    path: string;
	    url: string;
	    sha1?: string;
	    size?: number;
	
	    static createFrom(source: any = {}) {
	        return new ModpackFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.url = source["url"];
	        this.sha1 = source["sha1"];
	        this.size = source["size"];
	    }
	}
	export class ModrinthProject {
	    project_id: string;
	    slug: string;
	    title: string;
	    description: string;
	    icon_url: string;
	    author: string;
	    downloads: number;
	    follows: number;
	    categories: string[];
	    versions: string[];
	    client_side: string;
	    server_side: string;
	    is_installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModrinthProject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project_id = source["project_id"];
	        this.slug = source["slug"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.icon_url = source["icon_url"];
	        this.author = source["author"];
	        this.downloads = source["downloads"];
	        this.follows = source["follows"];
	        this.categories = source["categories"];
	        this.versions = source["versions"];
	        this.client_side = source["client_side"];
	        this.server_side = source["server_side"];
	        this.is_installed = source["is_installed"];
	    }
	}
	export class ModrinthSearchResult {
	    hits: ModrinthProject[];
	    total_hits: number;
	
	    static createFrom(source: any = {}) {
	        return new ModrinthSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], ModrinthProject);
	        this.total_hits = source["total_hits"];
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
	export class ServerLiveStatus {
	    online: boolean;
	    pingMs: number;
	    players: number;
	    maxPlayers: number;
	    version: string;
	    motd: string;
	    favicon?: string;
	    playerList?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ServerLiveStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.online = source["online"];
	        this.pingMs = source["pingMs"];
	        this.players = source["players"];
	        this.maxPlayers = source["maxPlayers"];
	        this.version = source["version"];
	        this.motd = source["motd"];
	        this.favicon = source["favicon"];
	        this.playerList = source["playerList"];
	    }
	}
	export class ServerManifest {
	    id: string;
	    name: string;
	    tagline: string;
	    description: string;
	    version: string;
	    changelog: string;
	    gameVersion: string;
	    loader: string;
	    loaderVersion?: string;
	    serverIp?: string;
	    serverPort?: number;
	    bannerUrl?: string;
	    minMemoryMb?: number;
	    maxMemoryMb?: number;
	    archive?: ModpackArchive;
	    files: ModpackFile[];
	
	    static createFrom(source: any = {}) {
	        return new ServerManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tagline = source["tagline"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.changelog = source["changelog"];
	        this.gameVersion = source["gameVersion"];
	        this.loader = source["loader"];
	        this.loaderVersion = source["loaderVersion"];
	        this.serverIp = source["serverIp"];
	        this.serverPort = source["serverPort"];
	        this.bannerUrl = source["bannerUrl"];
	        this.minMemoryMb = source["minMemoryMb"];
	        this.maxMemoryMb = source["maxMemoryMb"];
	        this.archive = this.convertValues(source["archive"], ModpackArchive);
	        this.files = this.convertValues(source["files"], ModpackFile);
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
	export class SystemStats {
	    cpuModel: string;
	    cpuCores: number;
	    totalRamMb: number;
	    freeRamMb: number;
	    gpuModel: string;
	    javaRuntimes: JavaRuntime[];
	    launcherDir: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuModel = source["cpuModel"];
	        this.cpuCores = source["cpuCores"];
	        this.totalRamMb = source["totalRamMb"];
	        this.freeRamMb = source["freeRamMb"];
	        this.gpuModel = source["gpuModel"];
	        this.javaRuntimes = this.convertValues(source["javaRuntimes"], JavaRuntime);
	        this.launcherDir = source["launcherDir"];
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
	export class UpdateCheckResult {
	    serverId: string;
	    serverName: string;
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    changelog: string;
	    filesToDownload: ModpackFile[];
	    filesToDelete: string[];
	    totalDownloadSize: number;
	    manifest?: ServerManifest;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.serverName = source["serverName"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.changelog = source["changelog"];
	        this.filesToDownload = this.convertValues(source["filesToDownload"], ModpackFile);
	        this.filesToDelete = source["filesToDelete"];
	        this.totalDownloadSize = source["totalDownloadSize"];
	        this.manifest = this.convertValues(source["manifest"], ServerManifest);
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
	export class VersionItem {
	    id: string;
	    type: string;
	    url: string;
	    // Go type: time
	    time: any;
	    // Go type: time
	    releaseTime: any;
	
	    static createFrom(source: any = {}) {
	        return new VersionItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.url = source["url"];
	        this.time = this.convertValues(source["time"], null);
	        this.releaseTime = this.convertValues(source["releaseTime"], null);
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
	export class VersionManifest {
	    latest: LatestVersion;
	    versions: VersionItem[];
	
	    static createFrom(source: any = {}) {
	        return new VersionManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.latest = this.convertValues(source["latest"], LatestVersion);
	        this.versions = this.convertValues(source["versions"], VersionItem);
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

export namespace tunnel {
	
	export class TunnelStatus {
	    running: boolean;
	    localPort: number;
	    serverTarget: string;
	    tailscaleIp: string;
	    statusText: string;
	    latencyMs: number;
	    activeConns: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TunnelStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.localPort = source["localPort"];
	        this.serverTarget = source["serverTarget"];
	        this.tailscaleIp = source["tailscaleIp"];
	        this.statusText = source["statusText"];
	        this.latencyMs = source["latencyMs"];
	        this.activeConns = source["activeConns"];
	        this.error = source["error"];
	    }
	}

}

