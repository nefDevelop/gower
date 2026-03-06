import QtQuick
import QtQuick.Controls
import qs.Widgets
import qs.Common

Rectangle {
    id: root
    width: parent.width
    height: 48
    radius: Theme.cornerRadius
    color: Theme.surfaceContainerHighest

    property var colors: []
    property string selectedColor: ""

    signal colorClicked(string color)

    Row {
        anchors.fill: parent
        objectName: "colorRow"
        Repeater {
            model: root.colors
            Rectangle {
                id: colorRect
                width: parent.width / Math.max(1, root.colors.length)
                height: parent.height
                color: modelData
                objectName: "colorRect_" + index
                
                property bool isFirst: index === 0
                property bool isLast: index === root.colors.length - 1
                property bool isOnly: root.colors.length === 1

                radius: (isFirst || isLast) ? Theme.cornerRadius : 0

                Rectangle {
                    visible: parent.isFirst && !parent.isOnly
                    width: parent.radius
                    anchors { right: parent.right; top: parent.top; bottom: parent.bottom }
                    color: parent.color
                }

                Rectangle {
                    visible: parent.isLast && !parent.isOnly
                    width: parent.radius
                    anchors { left: parent.left; top: parent.top; bottom: parent.bottom }
                    color: parent.color
                }

                Rectangle {
                    anchors.fill: parent
                    color: "#000000"
                    opacity: root.selectedColor === modelData ? 0.4 : 0
                    Behavior on opacity { NumberAnimation { duration: 150 } }
                    radius: parent.radius
                    layer.enabled: true

                    Rectangle {
                        visible: colorRect.isFirst && !colorRect.isOnly
                        width: parent.radius
                        anchors { right: parent.right; top: parent.top; bottom: parent.bottom }
                        color: "#000000"
                    }

                    Rectangle {
                        visible: colorRect.isLast && !colorRect.isOnly
                        width: parent.radius
                        anchors { left: parent.left; top: parent.top; bottom: parent.bottom }
                        color: "#000000"
                    }
                }

                DankIcon {
                    anchors.centerIn: parent
                    name: "check"
                    size: 20
                    color: "white"
                    visible: root.selectedColor === modelData
                    opacity: visible ? 1 : 0
                    Behavior on opacity { NumberAnimation { duration: 150 } }
                }
                
                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.colorClicked(modelData)
                }
            }
        }
    }
}
