import QtQuick
import QtQuick.Controls 6.2
import QtQuick.Layouts 6.2
import qs.Modules.Plugins
import qs.Common // Para Theme
import Quickshell.Io // Importamos Quickshell.Io (sin versión específica)

PluginComponent {
    id: root

    // Popout sizing, can be adjusted
    popoutWidth: 350
    popoutHeight: 280

    // QML Backend Object
    PluginComponent {
        id: backend
        property string current_mode: "Cargando..."
        property int timer_minutes: 0
        property string wallpaper_folder: "Cargando..."

        signal statusChanged()

        // Corregir la ruta del script Python según el feedback del usuario.
        // Asumiendo que pluginData.scriptPath es la fuente original o preferida.
        // Añadimos Qt.resolvedPath como fallback si pluginData.scriptPath no está disponible.
        readonly property string scriptPath: root.pluginData.scriptPath || Qt.resolvedPath("backend.py")

        // Component para crear instancias del proceso dinámicamente
        Component {
            id: scriptProcessComponent

            Process {
                // Inicializamos la propiedad 'command' con un array vacío para satisfacer
                // el posible requisito de propiedad por defecto.
                command: [] 

                stdout: SplitParser {
                    onRead: line => {
                        backend.processPythonOutput(line);
                    }
                }
                stderr: SplitParser {
                    onRead: line => {
                        if (line.trim()) {
                            console.error("Python Backend Error (stderr): " + line);
                        }
                    }
                }

            }
        }

        function _executeCommand(command, args = []) {
            // Crear una nueva instancia del Process cada vez
            var process = scriptProcessComponent.createObject(root);
            process.command = ["python", backend.scriptPath, command].concat(args);
            process.running = true; // Iniciar el proceso
        }

        function getStatus() {
            _executeCommand("status")
        }

        function setNow() {
            _executeCommand("next")
        }

        function toggleMode() {
            _executeCommand("toggle")
        }

        function setTimer(minutes) {
            _executeCommand("set-config", ["'{\"timer_minutes\": " + minutes + "}'"])
        }

        // Función para procesar la salida JSON del script Python (recibida de SplitParser)
        function processPythonOutput(output) {
            // SplitParser puede emitir líneas vacías o parciales, filtrar.
            if (!output.trim()) return;

            try {
                var data = JSON.parse(output);
                if (data.error) {
                    console.error("Backend Error: " + data.error);
                    backend.current_mode = "ERROR";
                    backend.wallpaper_folder = data.error;
                } else {
                    backend.current_mode = data.current_mode || "desconocido";
                    backend.timer_minutes = data.timer_minutes || 0;
                    var folder = data.wallpaper_folder || "No configurada";
                    backend.wallpaper_folder = Qt.basename(folder);
                }
                backend.statusChanged(); // Emitir señal para actualizar la UI
            } catch (e) {
                console.error("Error parsing Python output: " + e + "\nOutput: " + output);
                backend.current_mode = "JSON ERROR";
                backend.wallpaper_folder = "Check console for invalid JSON";
            }
        }

        Component.onCompleted: {
            backend.getStatus(); // Obtener el estado inicial al cargar el componente
        }
    }

    // Temporizador para refrescar el estado periódicamente
    Timer {
        interval: 5000 // Cada 5 segundos
        running: true
        repeat: true
        onTriggered: backend.getStatus()
    }

    // Conectar la señal de cambio del backend QML a una función de log (para depuración)
    Connections {
        target: backend
        function onStatusChanged() {
            console.log("QML Backend status changed. Mode: " + backend.current_mode + ", Folder: " + backend.wallpaper_folder);
        }
    }

    // --- Pill para la barra horizontal (solo texto simple) ---
    horizontalBarPill: Component {
        Rectangle {
            implicitWidth: 30
            implicitHeight: 30
            color: Theme.surfaceContainerHigh

            Row {
                anchors.centerIn: parent
                spacing: Theme.spacingXS
                Text {
                    text: "WP"
                    font.pixelSize: Theme.fontSizeLarge
                    color: "white" // Color fijo para asegurar visibilidad
                }
            }
        }
    }

    // --- Pill para la barra vertical (solo texto simple) ---
    verticalBarPill: Component {
        Rectangle {
            implicitWidth: 30
            implicitHeight: 30
            color: Theme.surfaceContainerHigh

            Column {
                anchors.centerIn: parent
                spacing: Theme.spacingXS
                Text {
                    text: "WP"
                    font.pixelSize: Theme.fontSizeMedium
                    color: "white" // Color fijo para asegurar visibilidad
                }
            }
        }
    }

    // --- Contenido del Popout (la UI completa del gestor) ---
    popoutContent: Component {
        PopoutComponent {
            id: popoutRoot

            headerText: "Gestor de Fondos"
            detailsText: "Controla tu fondo de pantalla"
            showCloseButton: true

            Item { // Contenedor directo del contenido
                width: parent.width
                implicitHeight: root.popoutHeight - popoutRoot.headerHeight - popoutRoot.detailsHeight - Theme.spacingXL

                // Contenido principal de la UI
                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 15
                    spacing: 10

                    // Fila de Estado
                    RowLayout {
                        Text {
                            id: modeText
                            text: "Modo: " + backend.current_mode
                            color: "white"
                            font.pixelSize: 16
                        }
                    }

                    Text {
                        id: folderText
                        text: "Carpeta: " + backend.wallpaper_folder
                        color: "#aab0b6"
                        font.pixelSize: 12
                        elide: Text.ElideRight
                    }

                    // Separador
                    Rectangle {
                        Layout.fillWidth: true
                        height: 1
                        color: "#373e48"
                    }

                    // Configuración del Temporizador
                    RowLayout {
                        Layout.fillWidth: true
                        Text {
                            text: "Cambiar cada (min):"
                            color: "white"
                            Layout.alignment: Qt.AlignVCenter
                        }
                        TextField {
                            id: timerInput
                            text: backend.timer_minutes.toString()
                            color: "white"
                            background: Rectangle { color: "#1c2128" }
                            horizontalAlignment: Text.AlignHCenter
                            validator: IntValidator { bottom: 1 }
                            Layout.fillWidth: true
                            onAccepted: backend.setTimer(parseInt(text))
                            enabled: backend.current_mode !== "Cargando..." && backend.current_mode !== "ERROR" && backend.current_mode !== "JSON ERROR" && backend.current_mode !== "ERROR (Python)" && backend.current_mode !== "ERROR (Process)"
                        }
                    }

                    // Botones de Acción
                    GridLayout {
                        columns: 2
                        Layout.fillWidth: true
                        columnSpacing: 10
                        rowSpacing: 10

                        Button {
                            text: "Cambiar Modo"
                            Layout.fillWidth: true
                            onClicked: backend.toggleMode()
                            enabled: backend.current_mode !== "Cargando..." && backend.current_mode !== "ERROR" && backend.current_mode !== "JSON ERROR" && backend.current_mode !== "ERROR (Python)" && backend.current_mode !== "ERROR (Process)"
                        }
                        Button {
                            text: "Forzar Cambio"
                            Layout.fillWidth: true
                            onClicked: backend.setNow()
                            enabled: backend.current_mode !== "Cargando..." && backend.current_mode !== "ERROR" && backend.current_mode !== "JSON ERROR" && backend.current_mode !== "ERROR (Python)" && backend.current_mode !== "ERROR (Process)"
                        }
                    }

                    // Botón para elegir carpeta (funcionalidad futura)
                    Button {
                        text: "Elegir Carpeta..."
                        Layout.fillWidth: true
                        enabled: false
                    }
                }
            }
        }
    }
}
