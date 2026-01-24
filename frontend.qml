import QtQuick 6.2
import QtQuick.Controls 6.2
import QtQuick.Layouts 6.2

ApplicationWindow {
    id: root
    width: 350
    height: 280
    title: "Gestor de Fondos"
    color: "#22272e"

    // Conectar la señal de cambio del backend a una función de log
    Connections {
        target: backend
        function onStatusChanged() {
            console.log("Backend status changed. Mode: " + backend.current_mode);
        }
    }

    // --- UI ---
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 15
        spacing: 10

        // Fila de Estado
        RowLayout {
            Text {
                id: modeText
                // Enlazar directamente a la propiedad del backend
                text: "Modo: " + backend.current_mode
                color: "white"
                font.pixelSize: 16
            }
        }

        Text {
            id: folderText
            // Enlazar directamente a la propiedad del backend
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
                // Enlazar a la propiedad del backend
                text: backend.timer_minutes.toString()
                color: "white"
                background: Rectangle { color: "#1c2128" }
                horizontalAlignment: Text.AlignHCenter
                validator: IntValidator { bottom: 1 }
                Layout.fillWidth: true
                // Llamar al slot del backend al aceptar
                onAccepted: backend.setTimer(parseInt(text))
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
                // Llamar al slot del backend
                onClicked: backend.toggleMode()
            }
            Button {
                text: "Forzar Cambio"
                Layout.fillWidth: true
                // Llamar al slot del backend
                onClicked: backend.setNow()
            }
        }
        
        // Botón para elegir carpeta (funcionalidad futura)
        Button {
            text: "Elegir Carpeta..."
            Layout.fillWidth: true
            enabled: false // Deshabilitado por ahora
        }
    }
}
