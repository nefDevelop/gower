pragma Singleton
import QtQml
import Quickshell
import Quickshell.Io

QtObject {
    id: backend

    signal colorsChanged(var feedColors, var favoritesColors)
    signal feedChanged(var feed)
    signal favoritesChanged(var favorites)
    signal searchChanged(var searchResult)
    signal feedNeedsReload()
    signal favoritesNeedReload()
    signal updateFinished()

    property string userHome: ""
    property int feedPage: 1
    property int feedLimit: 9
    property string cachePath: ""
    property string configPath: ""
    property string dataPath: ""
    property bool refreshing: false
    property bool initialized: false
    property bool isLaptop: false
    property var monitors: []
    property int feedRetryCount: 0
    property string currentFeedColor: ""
    property bool daemonRunning: false
    property var currentWallpapers: []
    property var currentWallpaperItems: []
    property string feedSort: "smart"
    property var config: null

    function getHsl(hex) {
        var result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
        if (!result) return {h:0, s:0, l:0};
        var r = parseInt(result[1], 16) / 255;
        var g = parseInt(result[2], 16) / 255;
        var b = parseInt(result[3], 16) / 255;
        var max = Math.max(r, g, b), min = Math.min(r, g, b);
        var h, s, l = (max + min) / 2;
        if (max == min) {
            h = s = 0;
        } else {
            var d = max - min;
            s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
            switch (max) {
                case r: h = (g - b) / d + (g < b ? 6 : 0); break;
                case g: h = (b - r) / d + 2; break;
                case b: h = (r - g) / d + 4; break;
            }
            h /= 6;
        }
        return {h: h, s: s, l: l};
    }

    property Component processComponent: Component {
        Process {
        }
    }

    property Component collectorComponent: Component {
        StdioCollector {}
    }

    function createProcess(command, onFinished) {
        var process = backend.processComponent.createObject(backend, {
            "command": command
        });
        var collector = backend.collectorComponent.createObject(process);
        process.stdout = collector;
        collector.onStreamFinished.connect(function() {
            onFinished(collector.text);
            process.destroy();
        });
        process.running = true;
    }

    function openImageExternally(item) {
        if (!backend.config || !backend.config.paths || !backend.config.paths.wallpapers) {
            console.error("Cannot open image externally, wallpaper path not configured.");
            return;
        }
        if (!item || !item.id || !item.ext) {
            console.error("Cannot open image externally, invalid item data.");
            return;
        }
        var imagePath = backend.config.paths.wallpapers + "/" + item.id + item.ext;
        // Use xdg-open to open with default application
        var cmd = ["xdg-open", imagePath];
        console.log("Opening image with command:", JSON.stringify(cmd));
        Quickshell.execDetached(cmd);
    }

    function openInBrowser(item) {
        if (!item) return;
        var url = item.permalink || item.post_url || item.url || item.link || "";
        if (url !== "") {
            console.error("Backend: Found direct URL in item, opening: " + url);
            Quickshell.execDetached(["xdg-open", url]);
            return;
        }

        // If no direct URL, try fetching details with gower
        if (!item.id) {
            console.error("Backend: No URL and no ID to fetch details for item.");
            console.error("Backend: Item data: " + JSON.stringify(item));
            return;
        }

        console.warn("Backend: No direct URL found for " + item.id + ". Fetching details with 'gower wallpaper'...");
        var cmd = ["gower", "wallpaper", item.id, "--json"];
        backend.createProcess(cmd, function(text) {
            var output = text.trim();
            if (output === "") {
                console.error("Backend: 'gower wallpaper' command returned empty output for " + item.id);
                return;
            }

            var jsonStart = output.indexOf("{");
            var jsonEnd = output.lastIndexOf("}");
            if (jsonStart === -1 || jsonEnd === -1) {
                console.error("Backend: Could not find JSON object in 'gower wallpaper' output for " + item.id + ". Raw: " + output);
                return;
            }

            try {
                var details = JSON.parse(output.substring(jsonStart, jsonEnd + 1));
                var fetchedUrl = details.permalink || details.post_url || details.url || details.link || "";
                if (fetchedUrl !== "") {
                    console.error("Backend: Fetched details, opening URL: " + fetchedUrl);
                    Quickshell.execDetached(["xdg-open", fetchedUrl]);
                } else {
                    console.error("Backend: Fetched details for " + item.id + " but still no URL found. Data: " + JSON.stringify(details));
                }
            } catch (e) {
                console.error("Backend: Failed to parse JSON from 'gower wallpaper' for " + item.id + ". Error: " + e + ". Raw: " + output);
            }
        });
    }

    function openImageFolder(item) {
        if (!backend.config || !backend.config.paths || !backend.config.paths.wallpapers) {
            console.error("Cannot open folder, wallpaper path not configured.");
            return;
        }
        var folderPath = backend.config.paths.wallpapers;
        var cmd = ["xdg-open", folderPath];
        Quickshell.execDetached(cmd);
    }

    function deleteWallpaper(id) {
        var cmd = ["gower", "wallpaper", id, "--delete", "--file", "--force"];
        backend.createProcess(cmd, function() {
            // After deleting, refresh the current wallpapers and the feed
            backend.loadCurrentWallpapers();
            backend.feedNeedsReload();
        });
    }

    function detectPaths(retry) {
        var xdgConfig = backend.userHome + "/.config/gower";
        var legacyConfig = backend.userHome + "/.gower";
        
        var cmd = "if [ -f \"" + xdgConfig + "/config.json\" ]; then echo XDG; " +
                  "elif [ -f \"" + legacyConfig + "/config.json\" ]; then echo LEGACY; " +
                  "elif [ -d \"" + xdgConfig + "\" ]; then echo XDG; " +
                  "elif [ -d \"" + legacyConfig + "\" ]; then echo LEGACY; " +
                  "else echo NONE; fi";

        backend.createProcess(["sh", "-c", cmd], function(output) {
            var mode = output.trim();
            console.log("Path detection mode: " + mode);

            if (mode === "XDG") {
                backend.configPath = xdgConfig;
                backend.dataPath = backend.userHome + "/.local/share/gower";
                backend.cachePath = backend.userHome + "/.cache/gower";
                backend.finalizeInit();
            } else if (mode === "LEGACY") {
                backend.configPath = legacyConfig;
                backend.dataPath = legacyConfig + "/data";
                backend.cachePath = legacyConfig + "/cache";
                backend.finalizeInit();
            } else {
                if (retry) {
                    console.error("Could not detect config paths even after 'gower config init'.");
                    return;
                }
                console.log("No configuration found. Running 'gower config init'...");
                backend.createProcess(["gower", "config", "init"], function() {
                    backend.detectPaths(true);
                });
            }
        });
    }

    function finalizeInit() {
        backend.loadConfig();
        backend.checkAndLoadColors();
        backend.loadCurrentWallpapers();
        backend.update();
    }

    function initialize() {
        if (backend.initialized) return;
        backend.initialized = true;
        backend.createProcess(["sh", "-c", "ls /sys/class/power_supply/BAT* > /dev/null 2>&1 && echo 1 || echo 0"], function(output) {
            backend.isLaptop = (output.trim() === "1");
        });
        backend.checkDaemonStatus();
        backend.loadMonitors();
        backend.createProcess(["sh", "-c", "echo $HOME"], function(text) {
            backend.userHome = text.trim();
            backend.detectPaths(false);
        });
    }

    function checkDaemonStatus() {
        backend.createProcess(["sh", "-c", "pgrep -f \"gower [d]aemon\" > /dev/null && echo 1 || echo 0"], function(output) {
            var status = (output.trim() === "1");
            console.error("Daemon status check (pgrep): " + status);
            backend.daemonRunning = status;
        });
    }

    function toggleDaemon(enable) {
        backend.daemonRunning = enable;
        if (enable) {
            backend.createProcess(["sh", "-c", "gower daemon start 2>&1"], function(output) {
                console.error("Daemon start output: " + output);
                var timer = Qt.createQmlObject("import QtQuick; Timer { interval: 2000; repeat: false; onTriggered: backend.checkDaemonStatus(); }", backend);
                timer.start();
            });
        } else {
            backend.createProcess(["sh", "-c", "(gower daemon stop; sleep 1; pkill -f \"gower daemon\") 2>&1"], function(output) {
                console.error("Daemon stop output: " + output);
                var timer = Qt.createQmlObject("import QtQuick; Timer { interval: 2000; repeat: false; onTriggered: backend.checkDaemonStatus(); }", backend);
                timer.start();
            });
        }
    }

    function loadMonitors() {
        backend.createProcess(["sh", "-c", "gower status --monitors --json 2>&1"], function(text) {
            var output = text.trim();
            if (output === "") {
                return;
            }
            var jsonStart = output.indexOf("{");
            var jsonEnd = output.lastIndexOf("}");
            
            if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
                output = output.substring(jsonStart, jsonEnd + 1);
            }
            try {
                var json = JSON.parse(output);
                if (json && json.monitors) {
                    backend.monitors = json.monitors;
                }
            } catch (e) {
                console.error("Error parsing monitors: " + e);
                console.error("Raw output: " + text);
            }
        });
    }

    function checkAndLoadColors() {
        if (backend.dataPath === "") return;
        var colorsFile = backend.dataPath + "/colors.json";
        backend.createProcess(["cat", colorsFile], function(text) {
            if (!text || text.trim() === "") {
                return
            }
            try {
                var data = JSON.parse(text)
                var sortFn = function(a, b) {
                    var hslA = backend.getHsl(a);
                    var hslB = backend.getHsl(b);
                    if (hslA.s < 0.01 && hslB.s >= 0.01) return 1;
                    if (hslA.s >= 0.01 && hslB.s < 0.01) return -1;
                    return hslA.h - hslB.h;
                }

                var feedColors = data.feed || data.feed_palette || [];
                var favColors = data.favorites || data.favorites_palette || [];

                if (feedColors.length > 0) {
                    feedColors.sort(sortFn)
                }
                if (favColors.length > 0) {
                    favColors.sort(sortFn)
                }
                backend.colorsChanged(feedColors, favColors);
            } catch (e) {
                console.error("Failed to parse colors JSON: " + e)
            }
        });
    }

    function update() {
        backend.createProcess(["gower", "feed", "update"], function(text) {
            backend.updateFinished();
            backend.checkAndLoadColors();
            backend.loadCurrentWallpapers(function() {
                backend.feedNeedsReload();
            });
        });
    }

    function undoWallpaper() {
        backend.createProcess(["gower", "set", "undo"], function(text) {
            backend.loadCurrentWallpapers(function() {
                backend.feedNeedsReload();
            });
        });
    }

    function loadFeed(color, isRetry) {
        if (!isRetry) {
            backend.feedRetryCount = 0;
            backend.currentFeedColor = color;
        }
        if (backend.userHome === "") {
            backend.feedChanged([]);
            return;
        }
        
        var limit = backend.feedLimit;
        
        var colorArg = (color && color !== "") ? " --color " + color.replace("#", "") : "";
        var sortArg = " --sort " + backend.feedSort;
        var feedCmd = "gower feed show --quiet --page " + backend.feedPage + " --limit " + limit + (backend.refreshing ? " --refresh" : "") + colorArg + sortArg + " --json 2>&1";
        
        backend.executeFeedCommand(feedCmd, isRetry);
    }

    function executeFeedCommand(commandStr, isRetry) {
        backend.createProcess(["sh", "-c", commandStr], function(text) {
            var output = text.trim();
            var items = [];
            
            if (output !== "") {
                var jsonStart = output.indexOf("[");
                var jsonEnd = output.lastIndexOf("]");
                
                if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
                    var jsonString = output.substring(jsonStart, jsonEnd + 1);
                    try {
                        items = JSON.parse(jsonString);
                    } catch (e) {
                        console.error("Feed JSON parse error: " + e);
                    }
                }
            }
            
            for (var i = 0; i < items.length; i++) {
                if (items[i].id && items[i].ext) {
                    items[i].thumbnail = "file://" + backend.cachePath + "/thumbs/" + items[i].id + items[i].ext;
                    if (isRetry) {
                        items[i].thumbnail += "?retry=" + new Date().getTime();
                    }
                } else {
                    items[i].thumbnail = "";
                }
                items[i].seen = true;
            }
            backend.validateFeedItems(items);
        });
    }

    function validateFeedItems(items) {
        if (!items || items.length === 0) {
            backend.feedChanged([]);
            return;
        }
        
        var idsToCheck = [];
        for (var i = 0; i < items.length; i++) {
            if (items[i].id && items[i].ext) {
                idsToCheck.push(items[i].id + "::" + items[i].ext);
            }
        }
        
        if (idsToCheck.length === 0) {
            backend.feedChanged(items);
            return;
        }

        var cmd = "for item in " + idsToCheck.join(" ") + "; do " +
                  "id=${item%%::*}; ext=${item##*::}; " +
                  "path=\"" + backend.cachePath + "/thumbs/$id$ext\"; " +
                  "if [ ! -s \"$path\" ] || ! file -b --mime-type \"$path\" | grep -q \"^image/\"; then echo $id; fi; " +
                  "done";

        backend.createProcess(["bash", "-c", cmd], function(output) {
            var missing = output.trim();
            if (missing) {
                console.error("Missing thumbnails detected for IDs: " + missing.replace(/\n/g, ", "));
            } 
            if (missing && backend.feedRetryCount < 3) {
                console.warn("Missing thumbnails detected. Running analysis and retrying (" + (backend.feedRetryCount + 1) + "/3)...");
                backend.feedRetryCount++;
                backend.createProcess(["gower", "feed", "analyze", "--all"], function(analyzeOutput) {
                    // Re-check if thumbnails are still missing after analyze
                    backend.createProcess(["bash", "-c", cmd], function(output2) {
                        var missing2 = output2.trim();
                        if (missing2) {
                             var ids = missing2.split("\n");
                             var downloadCmd = "for id in " + ids.join(" ") + "; do gower download $id; done";
                             console.warn("Thumbnails still missing after analyze. Downloading explicitly: " + ids.length);
                             backend.createProcess(["bash", "-c", downloadCmd], function() {
                                 backend.loadFeed(backend.currentFeedColor, true);
                             });
                        } else {
                             backend.loadFeed(backend.currentFeedColor, true);
                        }
                    });
                });
            } else {
                if (missing) {
                    console.warn("Giving up on missing thumbnails after retries.");
                    var missingList = missing.split("\n");
                    for (var m = 0; m < missingList.length; m++) {
                        var mId = missingList[m].trim();
                        if (!mId) continue;
                        for (var k = 0; k < items.length; k++) {
                            if (items[k].id === mId) items[k].thumbnail = "";
                        }
                    }
                }
                backend.feedChanged(items);
            }
        });
    }

    function loadFavorites(color) {
        if (backend.userHome === "") {
            backend.favoritesChanged([]);
            return;
        }
        var colorArg = (color && color !== "") ? " --color " + color.replace("#", "") : "";
        var command = ["sh", "-c", "gower favorites list" + colorArg + " --json 2>&1"];
        backend.createProcess(command, function(text) {
            var output = text.trim();
            if (output === "") {
                backend.favoritesChanged([]);
                return;
            }

            var jsonStart = output.indexOf("[");
            var jsonEnd = output.lastIndexOf("]");
            if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
                output = output.substring(jsonStart, jsonEnd + 1);
            }

            try {
                var favs = JSON.parse(output);
                var newModel = [];
                if (Array.isArray(favs)) {
                    newModel = favs.map(function(item) {
                        var favItem = (typeof item === 'object') ? item : { id: item };
                        if (favItem.id && favItem.hasOwnProperty('ext')) {
                            favItem.thumbnail = "file://" + backend.cachePath + "/thumbs/" + favItem.id + favItem.ext;
                        } else {
                            favItem.thumbnail = "";
                        }
                        favItem.seen = false;
                        return favItem;
                    });
                }
                backend.checkMissingThumbnails(newModel);
                backend.favoritesChanged(newModel);
            } catch (e) {
                console.error("Favorites JSON parse error: " + e);
                backend.favoritesChanged([]);
            }
        });
    }

    function loadCurrentWallpapers(callback) {
        backend.createProcess(["sh", "-c", "gower status --wallpapers --json 2>&1"], function(text) {
            var output = text.trim();
            var jsonStart = output.indexOf("{");
            var jsonEnd = output.lastIndexOf("}");
            
            var items = [];
            var ids = [];

            if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
                try {
                    var json = JSON.parse(output.substring(jsonStart, jsonEnd + 1));
                    // La nueva estructura es json.wallpaper.wallpapers, que es un array de objetos
                    if (json && json.wallpaper && json.wallpaper.wallpapers) {
                        items = json.wallpaper.wallpapers;
                        
                        // Procesamos los items para añadir la ruta local a la miniatura y extraer los IDs
                        for (var i = 0; i < items.length; i++) {
                            var item = items[i];
                            if (item.id && item.ext) {
                                item.thumbnail = "file://" + backend.cachePath + "/thumbs/" + item.id + item.ext;
                            }
                            ids.push(item.id);
                        }
                    }
                } catch (e) {
                    console.error("Error parsing wallpaper status: " + e);
                }
            }

            backend.currentWallpapers = ids;
            backend.currentWallpaperItems = items;
            console.error("Backend: Detected " + ids.length + " active wallpapers from gower status: " + JSON.stringify(ids));
            console.error("Backend: Resolved " + items.length + " wallpaper items from status.");

            if (callback) callback();
        });
    }

    function search(text, provider) {
        var providerKey = provider.toLowerCase();
        // This is a simplified version, in a real app you would get this from the config
        var cmd = ["sh", "-c", "gower explore \"" + text.replace(/"/g, '\\"') + "\" --provider \"" + providerKey + "\" --json 2>&1"];
        backend.createProcess(cmd, function(text) {
            var output = text.trim();
            if (output === "") {
                backend.searchChanged([]);
                return;
            }
            var jsonStart = output.indexOf("[");
            var jsonEnd = output.lastIndexOf("]");
            
            if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
                var jsonString = output.substring(jsonStart, jsonEnd + 1);
                try {
                    var items = JSON.parse(jsonString);
                    for (var i = 0; i < items.length; i++) {
                        if (!items[i].thumbnail) items[i].thumbnail = "";
                    }
                    backend.searchChanged(items);
                } catch (e) {
                    console.error("Search JSON parse error: " + e);
                    backend.searchChanged([]);
                }
            } else {
                backend.searchChanged([]);
            }
        });
    }

    function loadConfig() {
        if (backend.configPath === "") return;
        var configFile = backend.configPath + "/config.json";
        backend.createProcess(["cat", configFile], function(text) {
            var raw = text.trim();
            var start = raw.indexOf("{");
            var end = raw.lastIndexOf("}");
            
            if (start !== -1 && end !== -1) {
                var jsonStr = raw.substring(start, end + 1);
                try { 
                    var conf = JSON.parse(jsonStr);
                    backend.config = conf;
                    if (conf.behavior && conf.behavior.daemon_enabled === true && !backend.daemonRunning) {
                        console.log("Daemon enabled in config, starting...");
                        backend.toggleDaemon(true);
                    }
                } catch (e) {
                    console.error("Config parse error: " + e);
                }
            } else {
                console.error("No JSON object found in config file: " + configFile);
                console.error("Raw content from cat: '" + raw + "'");
                // Fallback to defaults to prevent UI breakage
                var defaults = {
                    paths: { wallpapers: backend.userHome + "/Pictures/Wallpapers", use_system_dir: false, index_wallpapers: false },
                    behavior: { change_interval: 30, auto_download: true, respect_dark_mode: true, multi_monitor: "clone" },
                    power: { pause_on_low_battery: true, low_battery_threshold: 20 },
                    providers: {},
                    generic_providers: {}
                };
                backend.config = defaults;
            }
        });
    }

    function setConfig(key, value) {
        console.error("Setting config: " + key + " = " + value);
        backend.createProcess(["gower", "config", "set", key + "=" + value], function() {
            backend.loadConfig();
        });
    }

    function addProvider(name, url, key) {
        var cmd = ["gower", "config", "provider", "add", name, url]
        if (key !== "") {
            cmd.push("--key");
            cmd.push(key);
        }
        createProcess(cmd, function() {
            backend.loadConfig();
        });
    }

    function addRedditProvider(channel, sort) {
        var cmd = ["gower", "config", "provider", "reddit", "add", channel, sort];
        backend.createProcess(cmd, function() {
            backend.loadConfig();
        });
    }

    function removeProvider(name) {
        var cmd = ["sh", "-c", "gower config provider remove \"" + name.replace(/"/g, '\\"') + "\""];
        backend.createProcess(cmd, function() {
            backend.loadConfig();
        });
    }

    function setWallpaper(id, monitor) {
        var cleanId = id;
        var qIndex = cleanId.indexOf('?');
        if (qIndex !== -1) {
            cleanId = cleanId.substring(0, qIndex);
        }
        var cmd = ["gower", "set", cleanId];
        if (monitor) {
            cmd.push("--target-monitor");
            cmd.push(monitor);
        }
        backend.createProcess(cmd, function() {
            backend.loadCurrentWallpapers();
        });
    }

    function blacklist(id) {
        var cmd = ["gower", "blacklist", "add", id]
        backend.createProcess(cmd, function() {
            backend.feedNeedsReload();
        });
    }

    function download(id) {
        var cleanId = id;
        var qIndex = cleanId.indexOf('?');
        if (qIndex !== -1) {
            cleanId = cleanId.substring(0, qIndex);
        }
        var cmd = ["gower", "download", cleanId];
        Quickshell.execDetached(cmd);
    }

    function checkMissingThumbnails(items) {
        if (!items || items.length === 0) return;
        
        var idsToCheck = [];
        for (var i = 0; i < items.length; i++) {
            if (items[i].id && items[i].ext) {
                idsToCheck.push(items[i].id + "::" + items[i].ext);
            }
        }
        
        if (idsToCheck.length === 0) return;

        var cmd = "for item in " + idsToCheck.join(" ") + "; do " +
                  "id=${item%%::*}; ext=${item##*::}; " +
                  "path=\"" + backend.cachePath + "/thumbs/$id$ext\"; " +
                  "if [ ! -s \"$path\" ] || ! file -b --mime-type \"$path\" | grep -q \"^image/\"; then echo $id; fi; " +
                  "done";

        backend.createProcess(["bash", "-c", cmd], function(output) {
            var missing = output.trim();
            if (missing) {
                var ids = missing.split("\n");
                console.error("Found " + ids.length + " missing thumbnails: " + ids.join(", "));
            } else {
                console.error("No missing or empty thumbnails detected in check.");
            }
        });
    }

    function handleImageError(id) {
        var cleanId = id;
        var qIndex = cleanId.indexOf('?');
        if (qIndex !== -1) {
            cleanId = cleanId.substring(0, qIndex);
        }
        console.error("FAILED TO LOAD IMAGE: " + cleanId);
    }

    function checkFile(path) {
        var cleanPath = path.replace("file://", "");
        backend.createProcess(["sh", "-c", "ls \"" + cleanPath + "\" > /dev/null 2>&1 && echo 1 || echo 0"], function(out) {
            console.log("File check " + cleanPath + ": " + (out.trim() === "1" ? "Exists" : "Missing"));
        });
    }

    function addFavorite(id) {
        var cmd = ["gower", "favorites", "add", id];
        backend.createProcess(cmd, function() {
            backend.feedNeedsReload();
            backend.favoritesNeedReload();
            backend.checkAndLoadColors();
        });
    }

    function removeFavorite(id) {
        var cmd = ["gower", "favorites", "remove", id];
        backend.createProcess(cmd, function() {
            backend.feedNeedsReload();
            backend.favoritesNeedReload();
            backend.checkAndLoadColors();
        });
    }

    function openFolderPicker(onFinished) {
        backend.createProcess(["zenity", "--file-selection", "--directory"], function(path) {
            if (path) {
                onFinished(path.trim());
            }
        });
    }
}