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
        const url = Qt.resolvedUrl("config.json").toString()
        return decodeURIComponent(url.startsWith("file://") ? url.substring(7) : url)
    }

    readonly property string indexScriptPath: {
        const url = Qt.resolvedUrl("wallpaperIndex.sh").toString()
        return decodeURIComponent(url.startsWith("file://") ? url.substring(7) : url)
    }

    readonly property string listScriptPath: {
        const url = Qt.resolvedUrl("wallpaperActual.sh").toString()
        return decodeURIComponent(url.startsWith("file://") ? url.substring(7) : url)
    }

    property string currentMode: (Qt.styleHints.colorScheme === Qt.ColorScheme.Light) ? "light" : "dark"
    onCurrentModeChanged: {
        root.refreshList();
    }

    property string wallpaperFolder: ""
    property int timerMinutes: 30
    property bool isLoading: false
    property bool isEditing: false
    property var wallpaperList: []
    property int currentIndex: 0
    property string currentWallpaper: ""
    property string nextWallpaper: ""
    property var lastSaveTime: 0

    // Icono basado en el modo
    property string displayIcon: "image"

    // --- Comunicación con Backend ---

    // Timer para actualizar el estado periódicamente
    Timer {
        interval: 5000
        running: true
        repeat: true
        onTriggered: {
            // Evitar leer si estamos editando, escribiendo, o si acabamos de guardar hace menos de 2s
            if (!root.isEditing && !writeConfigProcess.running && (Date.now() - root.lastSaveTime > 2000)) {
                readConfigProcess.running = true;
            }
        }
    }

    Component.onCompleted: {
        console.warn("WallpaperWidget: Loaded. Config Path: " + root.configPath);
        console.warn("WallpaperWidget: List Script: " + root.listScriptPath);
        // Retrasamos un poco el inicio para asegurar que todo esté cargado
        startupTimer.start();
    }

    Timer {
        id: startupTimer
        interval: 1000
        repeat: false
        onTriggered: readConfigProcess.running = true
    }

    // Proceso para leer la configuración
    Process {
        id: readConfigProcess
        command: ["cat", root.configPath]
        // onStarted: console.warn("WallpaperWidget: readConfigProcess started")
        // onExited: (code) => console.warn("WallpaperWidget: readConfigProcess exited with code " + code)
        stderr: StdioCollector { onStreamFinished: if(text.trim()) console.warn("WallpaperWidget: Read Config Error: " + text) }
        stdout: StdioCollector {
            onStreamFinished: {
                var data = text.trim();
                // console.warn("WallpaperWidget: Config data: " + data);
                if (!data) return;
                try {
                    var json = JSON.parse(data);
                    if (json.wallpaper_folder && !root.isEditing) root.wallpaperFolder = json.wallpaper_folder;
                    if (json.timer_minutes) root.timerMinutes = json.timer_minutes;
                    
                    if (root.wallpaperList.length === 0 && root.wallpaperFolder) {
                        root.refreshList();
                    }
                    // console.warn("WallpaperWidget: Config parsed. Folder: " + root.wallpaperFolder);
                } catch (e) {
                    console.warn("WallpaperWidget: Error parsing status JSON: " + e);
                }
            }
        }
    }

    // Proceso para guardar la configuración
    Process {
        id: writeConfigProcess
        command: []
    }

    // Proceso para ejecutar awww
    Process {
        id: awwwProcess
        command: [] 
        onStarted: console.warn("WallpaperWidget: Setting wallpaper...")
        stderr: StdioCollector { onStreamFinished: if(text.trim()) console.warn("WallpaperWidget: Set wallpaper error: " + text) }
        onExited: (code) => {
            root.isLoading = false;
            if (code === 0) console.warn("WallpaperWidget: Wallpaper set successfully.");
            else console.warn("WallpaperWidget: Failed to set wallpaper. Code: " + code);
        }
    }

    // Proceso para abrir carpeta
    Process {
        id: openFolderProcess
        command: []
    }

    // Proceso para borrar wallpaper
    Process {
        id: deleteProcess
        command: []
        onExited: (code) => {
            if (code === 0) {
                root.refreshList();
            }
        }
    }

    // Proceso para listar fondos
    Process {
        id: listProcess
        command: []
        stderr: StdioCollector { onStreamFinished: if(text.trim()) console.warn("WallpaperWidget: List Process Error: " + text) }
        stdout: StdioCollector {
            onStreamFinished: {
                var data = text.trim();
                // console.warn("WallpaperWidget: List Process Output (first 50 chars): " + data.substring(0, 50));
                if (!data) return;
                var lines = data.split('\n');
                var list = [];
                for (var i = 0; i < lines.length; i++) {
                    if (lines[i]) list.push(lines[i]);
                }
                // Mezclar lista
                for (let i = list.length - 1; i > 0; i--) {
                    const j = Math.floor(Math.random() * (i + 1));
                    [list[i], list[j]] = [list[j], list[i]];
                }
                root.wallpaperList = list;
                root.updateWallpapers();
            }
        }
    }

    // Proceso para indexar fondos (reemplaza lógica de Python)
    Process {
        id: indexProcess
        command: []
        onStarted: console.warn("WallpaperWidget: Indexing started...")
        stderr: StdioCollector { onStreamFinished: if(text.trim()) console.warn("WallpaperWidget: Index Process Error: " + text) }
        onExited: (code) => {
            root.isLoading = false;
            if (code !== 0) console.warn("Index process failed with code: " + code);
            root.refreshList();
        }
    }

    // Timer de seguridad por si el indexado se cuelga
    Timer {
        interval: 30000 // 30 segundos timeout
        running: root.isLoading
        onTriggered: {
            root.isLoading = false;
            root.refreshList();
        }
    }

    function saveConfig() {
        var data = {
            "current_mode": root.currentMode,
            "wallpaper_folder": root.wallpaperFolder,
            "timer_minutes": root.timerMinutes
        };
        var content = JSON.stringify(data, null, 4);
        var safeContent = content.replace(/'/g, "'\\''");
        var safePath = root.configPath.replace(/'/g, "'\\''");
        
        writeConfigProcess.command = ["sh", "-c", "printf '%s' '" + safeContent + "' > '" + safePath + "'"];
        root.lastSaveTime = Date.now();
        writeConfigProcess.running = true;
    }

    function refreshList() {
        if (!root.wallpaperFolder) return;
        // console.warn("WallpaperWidget: Refreshing list for folder: " + root.wallpaperFolder + " mode: " + root.currentMode);
        // Pasamos los argumentos directamente en el array, SIN escapar comillas (Process lo maneja)
        listProcess.command = ["sh", root.listScriptPath, root.wallpaperFolder, root.currentMode];
        listProcess.running = true;
    }

    function updateWallpapers() {
        if (root.wallpaperList.length > 0) {
            if (root.currentIndex >= root.wallpaperList.length) root.currentIndex = 0;
            root.currentWallpaper = root.wallpaperList[root.currentIndex];
            var nextIdx = (root.currentIndex + 1) % root.wallpaperList.length;
            root.nextWallpaper = root.wallpaperList[nextIdx];
        } else {
            root.currentWallpaper = "";
            root.nextWallpaper = "";
        }
    }

    function executeCommand(cmd, arg) {
        if (cmd === "next") {
            if (root.wallpaperList.length === 0) return;
            root.currentIndex = (root.currentIndex + 1) % root.wallpaperList.length;
            root.updateWallpapers();
            awwwProcess.command = ["dms", "ipc", "call", "wallpaper", "set", root.currentWallpaper];
            awwwProcess.running = true;
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
            root.isLoading = true;
            // Pasamos los argumentos directamente en el array, SIN escapar comillas
            indexProcess.command = ["sh", root.indexScriptPath, root.wallpaperFolder];
            indexProcess.running = true;
        } else if (cmd === "open-folder") {
            if (root.wallpaperFolder) {
                openFolderProcess.command = ["xdg-open", root.wallpaperFolder];
                openFolderProcess.running = true;
            }
        } else if (cmd === "delete") {
            if (root.currentWallpaper) {
                deleteProcess.command = ["rm", root.currentWallpaper];
                deleteProcess.running = true;
            }
        } else if (cmd === "set") {
            if (root.currentWallpaper) {
                awwwProcess.command = ["dms", "ipc", "call", "wallpaper", "set", root.currentWallpaper];
                awwwProcess.running = true;
            }
        }
    }

    // Timer para cambiar fondo automáticamente
    Timer {
        interval: root.timerMinutes * 60 * 1000
        running: root.timerMinutes > 0 && root.wallpaperList.length > 0
        repeat: true
        onTriggered: root.executeCommand("next")
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

    // --- Componentes UI Personalizados (para arreglar la interfaz extraña) ---

    component StyledButton: Rectangle {
        id: btn
        property string text: ""
        signal clicked()
        property bool enabled: true
        
        implicitWidth: 100
        implicitHeight: 36
        color: enabled ? (ma.containsMouse ? (Theme.surfaceContainerHighest || "#444444") : (Theme.surfaceContainerHigh || "#333333")) : (Theme.surfaceContainer || "#222222")
        radius: Theme.cornerRadius
        opacity: enabled ? 1.0 : 0.5

        StyledText {
            anchors.centerIn: parent
            text: btn.text
            font.weight: Font.Medium
            color: Theme.surfaceText || "#FFFFFF"
        }

        MouseArea {
            id: ma
            anchors.fill: parent
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            enabled: btn.enabled
            onClicked: btn.clicked()
        }
    }

    component StyledTextField: TextField {
        color: Theme.surfaceText || "#FFFFFF"
        placeholderTextColor: Theme.surfaceTextSecondary || "#AAAAAA"
        background: Rectangle {
            color: parent.activeFocus ? (Theme.surfaceContainerHighest || "#333333") : (Theme.surfaceContainer || "#222222")
            radius: Theme.cornerRadius || 4
            border.width: 1
            border.color: parent.activeFocus ? (Theme.primary || "#FFFFFF") : "transparent"
        }
        font.pixelSize: Theme.fontSizeMedium || 14
        leftPadding: 10
        rightPadding: 10
    }

    // --- Popout ---

    popoutWidth: 420
    popoutHeight: 600

    popoutContent: Component {
        PopoutComponent {
            id: popout

            headerText: "Gestor de Fondos"
            detailsText: "Controla tu fondo de pantalla"
            showCloseButton: true

            // Forzar lectura al abrir el popout
            onVisibleChanged: if (visible) readConfigProcess.running = true

            ColumnLayout {
                width: parent.width
                spacing: Theme.spacingM

                // Sección Carpeta
                Rectangle {
                    Layout.fillWidth: true
                    implicitHeight: folderCol.implicitHeight + Theme.spacingM * 2
                    radius: Theme.cornerRadius
                    color: Theme.surfaceContainerHigh
                    
                    ColumnLayout {
                        id: folderCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacingM
                        spacing: Theme.spacingXS
                        
                        RowLayout {
                            DankIcon { name: "folder"; size: Theme.iconSize - 4; color: Theme.primary || Theme.surfaceText || "#FFFFFF" }
                            StyledText { text: "Carpeta de Fondos"; font.weight: Font.Medium; color: Theme.surfaceVariantText }
                        }
                        
                        StyledTextField {
                            id: folderField
                            Layout.fillWidth: true
                            placeholderText: "/ruta/a/tus/wallpapers"
                            
                            Binding {
                                target: folderField
                                property: "text"
                                value: root.wallpaperFolder
                                when: !root.isEditing
                            }

                            onTextEdited: root.isEditing = true
                            onEditingFinished: {
                                root.wallpaperFolder = text;
                                root.saveConfig();
                                root.isEditing = false;
                                root.refreshList();
                            }
                        }
                    }
                }

                // Sección Wallpaper Actual
                RowLayout {
                    Layout.fillWidth: true
                    height: 180
                    spacing: Theme.spacingS

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        radius: Theme.cornerRadius
                        color: Theme.surfaceContainerHigh
                        clip: true

                        Image {
                            anchors.fill: parent
                            source: root.currentWallpaper ? "file://" + root.currentWallpaper : ""
                            fillMode: Image.PreserveAspectCrop
                            asynchronous: true
                            cache: true
                            visible: root.currentWallpaper !== ""
                        }
                        
                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.executeCommand("set")
                        }

                        ColumnLayout {
                            anchors.centerIn: parent
                            visible: !root.currentWallpaper
                            DankIcon { name: "image_not_supported"; size: 48; color: Theme.surfaceVariantText; Layout.alignment: Qt.AlignHCenter }
                            StyledText { text: "Sin wallpaper"; color: Theme.surfaceVariantText }
                        }
                    }
                    
                    ColumnLayout {
                        Layout.fillHeight: true
                        spacing: Theme.spacingS
                        
                        StyledButton {
                            text: "Abrir"
                            implicitWidth: 80
                            onClicked: root.executeCommand("open-folder")
                        }
                        
                        Item { Layout.fillHeight: true } // Espaciador

                        StyledButton {
                            text: "Borrar"
                            implicitWidth: 80
                            onClicked: root.executeCommand("delete")
                        }
                    }
                }

                // Controles
                RowLayout {
                    Layout.alignment: Qt.AlignHCenter
                    spacing: Theme.spacingM

                    StyledButton {
                        text: "Siguiente"
                        onClicked: root.executeCommand("next")
                    }

                    StyledButton {
                        text: "Indexar"
                        onClicked: root.executeCommand("index")
                        enabled: !root.isLoading && root.wallpaperFolder !== ""
                    }
                }

                RowLayout {
                    Layout.alignment: Qt.AlignHCenter
                    spacing: Theme.spacingS

                    StyledText { text: "Timer (min):" }
                    StyledTextField {
                        text: root.timerMinutes.toString()
                        implicitWidth: 60
                        onAccepted: root.executeCommand("set-config", JSON.stringify({timer_minutes: parseInt(text)}))
                    }
                }
            }
        }
    }
}
