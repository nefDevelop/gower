import QtQuick 2.15
import QtQuick.Controls 2.15
import qs.Common 1.0

TextField {
    id: control
    width: 200
    height: 40 // Altura mínima recomendada para etiqueta flotante interna

    property string label: ""
    placeholderText: "" // Desactivamos el placeholder nativo

    color: Theme.surfaceText
    selectionColor: Theme.primary
    selectedTextColor: Theme.onPrimary
    verticalAlignment: Text.AlignVCenter
    
    // Padding para dejar espacio a la etiqueta arriba
    topPadding: 16
    bottomPadding: 0
    leftPadding: 12
    rightPadding: 12

    background: Rectangle {
        color: control.activeFocus ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
        radius: 4
        border.width: control.activeFocus ? 2 : 1
        border.color: control.activeFocus ? Theme.primary : Theme.outline
        Behavior on border.color { ColorAnimation { duration: 150 } }
        Behavior on border.width { NumberAnimation { duration: 150 } }
    }

    Text {
        id: floatingLabel
        text: control.label
        
        property bool isFloating: control.activeFocus || control.text.length > 0
        
        x: control.leftPadding
        y: isFloating ? 2 : (control.height - height) / 2
        
        font.pixelSize: isFloating ? 10 : 14
        color: control.activeFocus ? Theme.primary : Theme.surfaceVariantText
        
        Behavior on y { NumberAnimation { duration: 150; easing.type: Easing.OutQuad } }
        Behavior on font.pixelSize { NumberAnimation { duration: 150 } }
        Behavior on color { ColorAnimation { duration: 150 } }
    }
}