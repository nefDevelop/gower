import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    // --- Configuración y Estado ---

    // Ruta al archivo de configuración
    readonly property string configPath: {
        const url = Qt.resolvedPath("config.json")
        return url.startsWith("file://") ? url.substring(7) : url
    }

    property string currentMode: "dark"
    property string wallpaperFolder: ""
    property int timerMinutes: 30
    property bool isLoading: false

    // Icono basado en el modo
    property string displayIcon: currentMode === "light" ? "weather-sunny" : "weather-night"

    // --- Comunicación con Backend ---

    // Timer para actualizar el estado periódicamente
    Timer {
        interval: 5000
        running: true
        repeat: true
        onTriggered: readConfigProcess.running = true
    }

    Component.onCompleted: {
        readConfigProcess.running = true
    }

    // Proceso para leer la configuración
    Process {
        id: readConfigProcess
        command: ["cat", root.configPath]
        stdout: SplitParser {
            onRead: data => {
                if (!data.trim()) return;
                try {
                    var json = JSON.parse(data);
                    if (json.current_mode) root.currentMode = json.current_mode;
                    if (json.wallpaper_folder) root.wallpaperFolder = json.wallpaper_folder;
                    if (json.timer_minutes) root.timerMinutes = json.timer_minutes;
                } catch (e) {
                    console.warn("WallpaperWidget: Error parsing status JSON: " + e);
                }
            }
        }
    }

    // Proceso para guardar la configuración
    Process {
        id: writeConfigProcess
        command: ["sh", "-c", "cat > '" + root.configPath + "'"]
        property string content: ""
        onStarted: {
            write(content);
            closeWrite();
        }
    }

    // Proceso para ejecutar awwww
    Process {
        id: awwwwProcess
        command: [] 
        onExited: (code) => {
            root.isLoading = false;
        }
    }

    // Proceso para indexar fondos (reemplaza lógica de Python)
    Process {
        id: indexProcess
        command: []
        onExited: (code) => {
            root.isLoading = false;
        }
    }

    function saveConfig() {
        var data = {
            "current_mode": root.currentMode,
            "wallpaper_folder": root.wallpaperFolder,
            "timer_minutes": root.timerMinutes
        };
        writeConfigProcess.content = JSON.stringify(data, null, 4);
        writeConfigProcess.running = true;
    }

    function executeCommand(cmd, arg) {
        if (cmd === "next") {
            awwwwProcess.command = ["awwww", "next"];
            awwwwProcess.running = true;
        } else if (cmd === "toggle") {
            root.currentMode = (root.currentMode === "light" ? "dark" : "light");
            saveConfig();
            var path = root.wallpaperFolder + "/" + root.currentMode;
            awwwwProcess.command = ["awwww", "set", path];
            awwwwProcess.running = true;
        } else if (cmd === "set-config") {
            try {
                var json = JSON.parse(arg);
                if (json.timer_minutes !== undefined) root.timerMinutes = json.timer_minutes;
                saveConfig();
            } catch (e) {
                console.error("Error parsing config arg: " + e);
            }
        } else if (cmd === "index") {
            if (!root.wallpaperFolder) return;
            // Escapar comillas simples en la ruta para el script de shell
            var safeDir = root.wallpaperFolder.replace(/'/g, "'\\''");
            
            // Script de shell equivalente a la función index_wallpapers de Python
            var script = "dir='" + safeDir + "'; " +
                         "mkdir -p \"$dir/light\" \"$dir/dark\"; " +
                         "rm -f \"$dir/light/\"* \"$dir/dark/\"*; " +
                         "find \"$dir\" -path \"$dir/light\" -prune -o -path \"$dir/dark\" -prune -o -type f \\( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' -o -iname '*.webp' \\) -print | while read -r img; do " +
                         "type=$(matugen -j \"$img\" | grep -o '\"type\":\"[^\"]*\"' | cut -d'\"' -f4); " +
                         "name=$(basename \"$img\"); " +
                         "if [ \"$type\" = \"light\" ]; then ln -sf \"$img\" \"$dir/light/$name\"; " +
                         "elif [ \"$type\" = \"dark\" ]; then ln -sf \"$img\" \"$dir/dark/$name\"; fi; " +
                         "done";
            
            indexProcess.command = ["sh", "-c", script];
            indexProcess.running = true;
        }
    }

    // --- Acciones del Plugin ---

    // --- Componentes Visuales (Pills) ---

    horizontalBarPill: Component {
        MouseArea {
            implicitWidth: contentRow.implicitWidth + Theme.spacingS
            implicitHeight: 30
            acceptedButtons: Qt.RightButton
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor

            onClicked: mouse => {
                if (mouse.button === Qt.RightButton) {
                    root.executeCommand("toggle");
                }
            }

            Row {
                id: contentRow
                anchors.centerIn: parent
                spacing: Theme.spacingXS

                DankIcon {
                    name: root.displayIcon
                    size: Theme.iconSize - 6
                    color: Theme.surfaceText
                }
            }
        }
    }

    verticalBarPill: Component {
        MouseArea {
            implicitWidth: 30
            implicitHeight: contentCol.implicitHeight + Theme.spacingS
            acceptedButtons: Qt.RightButton
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor

            onClicked: mouse => {
                if (mouse.button === Qt.RightButton) {
                    root.executeCommand("toggle");
                }
            }

            Column {
                id: contentCol
                anchors.centerIn: parent
                spacing: Theme.spacingXS

                DankIcon {
                    name: root.displayIcon
                    size: Theme.iconSize - 6
                    color: Theme.surfaceText
                }
            }
        }
    }

    // --- Popout ---

    popoutWidth: 320
    popoutHeight: 300

    popoutContent: Component {
        PopoutComponent {
            id: popout

            headerText: "Gestor de Fondos"
            detailsText: "Controla tu fondo de pantalla"
            showCloseButton: true

            ColumnLayout {
                width: parent.width
                spacing: Theme.spacingM

                StyledText {
                    text: "Carpeta: " + (root.wallpaperFolder || "No configurada")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.surfaceTextSecondary
                    Layout.fillWidth: true
                    elide: Text.ElideMiddle
                    horizontalAlignment: Text.AlignHCenter
                }

                RowLayout {
                    Layout.alignment: Qt.AlignHCenter
                    spacing: Theme.spacingL

                    Button {
                        text: "Siguiente"
                        onClicked: root.executeCommand("next")
                    }

                    Button {
                        text: "Modo: " + (root.currentMode === "light" ? "Claro" : "Oscuro")
                        onClicked: root.executeCommand("toggle")
                    }

                    Button {
                        text: "Indexar"
                        onClicked: root.executeCommand("index")
                        enabled: !root.isLoading && root.wallpaperFolder !== ""
                    }
                }

                RowLayout {
                    Layout.alignment: Qt.AlignHCenter
                    spacing: Theme.spacingS

                    StyledText { text: "Timer (min):" }
                    TextField {
                        text: root.timerMinutes.toString()
                        implicitWidth: 60
                        onAccepted: root.executeCommand("set-config", JSON.stringify({timer_minutes: parseInt(text)}))
                    }
                }
            }
        }
    }
}
