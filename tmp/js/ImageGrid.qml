import QtQuick
import QtQuick.Controls
import qs.Widgets
import qs.Common
import "."

Item {
    id: root

    property var model
    property var favoritesList: []
    property var currentWallpapers: []
    property alias count: grid.count
    property int limit: 9

    signal imageClicked(var modelData)
    signal blacklistClicked(var modelData)
    signal downloadClicked(var modelData)
    signal favoriteClicked(var modelData)

    GridView {
        id: grid
        anchors.fill: parent
        cellWidth: width / 3
        cellHeight: cellWidth
        model: root.model
        delegate: Item {
            width: GridView.view.cellWidth
            height: GridView.view.cellHeight


            Rectangle {
                anchors.fill: parent
                anchors.margins: 2
                color: Theme.surfaceContainerHighest
                radius: 4
                clip: true

                Image {
                    id: thumb
                    anchors.fill: parent
                    source: modelData.thumbnail || ""
                    fillMode: Image.PreserveAspectCrop
                    asynchronous: true
                    cache: false
                }

                // Active Wallpaper Indicator
                Rectangle {
                    anchors.top: parent.top
                    anchors.right: parent.right
                    anchors.margins: 6
                    width: 24
                    height: 24
                    radius: 12
                    color: Theme.primary
                    visible: root.currentWallpapers && root.currentWallpapers.indexOf(modelData.id) !== -1
                    z: 5

                    DankIcon {
                        anchors.centerIn: parent
                        name: "check"
                        size: 16
                        color: Theme.onPrimary
                    }
                }

                DankIcon {
                    anchors.centerIn: parent
                    name: "image"
                    size: 24
                    color: Theme.surfaceVariantText
                    visible: thumb.status !== Image.Ready
                }

                Rectangle {
                    anchors.fill: parent
                    gradient: Gradient {
                        GradientStop { position: 0.0; color: "transparent" }
                        GradientStop { position: 0.4; color: "transparent"}
                        GradientStop { position: 1.0; color: "#000000" }
                    }
                    opacity: hoverHandler.hovered ? 0.6 : 0
                    Behavior on opacity { NumberAnimation { duration: 200 } }
                }

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                    onClicked: (mouse) => {
                        if (mouse.button === Qt.RightButton) {
                            console.warn("ImageGrid: Right click - opening in browser for item: " + (modelData.id || "unknown"));
                            Backend.openInBrowser(modelData);
                        } else {
                            root.imageClicked(modelData);
                        }
                    }
                }

                Row {
                    anchors.bottom: parent.bottom
                    anchors.horizontalCenter: parent.horizontalCenter
                    anchors.bottomMargin: 8
                    spacing: 16
                    z: 10
                    visible: hoverHandler.hovered

                    DankIcon {
                        name: "visibility_off"
                        size: 20
                        color: "white"
                        opacity: maHide.containsMouse ? 1.0 : 0.6
                        Behavior on opacity { NumberAnimation { duration: 150 } }
                        MouseArea {
                            id: maHide
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.blacklistClicked(modelData)
                        }
                    }
                    DankIcon {
                        name: "download"
                        size: 20
                        color: "white"
                        opacity: maDownload.containsMouse ? 1.0 : 0.6
                        Behavior on opacity { NumberAnimation { duration: 150 } }
                        MouseArea {
                            id: maDownload
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.downloadClicked(modelData)
                        }
                    }
                    DankIcon {
                        id: favIcon
                        property bool isFav: root.favoritesList.indexOf(modelData.id) !== -1
                        name: isFav ? "favorite" : "favorite_border"
                        size: 20
                        color: isFav ? "red" : "white"
                        opacity: isFav ? 1.0 : (maFav.containsMouse ? 1.0 : 0.6)
                        Behavior on opacity { NumberAnimation { duration: 150 } }
                        Behavior on color { ColorAnimation { duration: 200 } }
                        MouseArea {
                            id: maFav
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.favoriteClicked(modelData)
                        }
                    }
                }

                HoverHandler {
                    id: hoverHandler
                    cursorShape: Qt.PointingHandCursor
                }
            }
        }
    }
}
