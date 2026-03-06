import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Widgets
import qs.Common
import "../js"

Item {
    id: tabHome
    width: parent.width
    height: 610
    implicitHeight: height

    property var feedColors: []
    property string selectedColor: ""
    property int feedPage: 1
    property var feedModel: null
    property var favoritesList: []
    property var currentWallpapers: []
    property var currentWallpaperItems: []
    property int feedLimit: 9
    property bool loading: false
    property var displayedItems: []

    signal loadFeed()
    signal colorClicked(string color)
    signal imageClicked(var modelData)
    signal blacklistClicked(var modelData)
    signal downloadClicked(var modelData)
    signal favoriteClicked(var modelData)
    signal pageChanged(int newPage)

    Timer {
        interval: 5000
        running: tabHome.parent && tabHome.parent.visible
        repeat: true
        onTriggered: Backend.loadCurrentWallpapers()
    }

    Component.onCompleted: {
        if (tabHome.currentWallpaperItems) {
            tabHome.displayedItems = tabHome.currentWallpaperItems
        }
    }

    onCurrentWallpaperItemsChanged: {
        var newItems = tabHome.currentWallpaperItems || []
        var currentDisplay = tabHome.displayedItems || []
        var result = []
        var maxLen = Math.max(newItems.length, currentDisplay.length)

        for (var i = 0; i < maxLen; i++) {
            var newItem = newItems[i]
            var oldItem = currentDisplay[i]

            if (oldItem && oldItem._isDeleted) {
                if (newItem && newItem.id !== oldItem.id) {
                    result.push(newItem)
                } else {
                    result.push(oldItem)
                }
            } else if (newItem) {
                result.push(newItem)
            }
        }
        tabHome.displayedItems = result
    }

    ColorFilter {
        id: colorFilter
        anchors.top: parent.top
        anchors.topMargin: Theme.spacingS
        colors: tabHome.feedColors
        selectedColor: tabHome.selectedColor
        onColorClicked: (color) => tabHome.colorClicked(color)
    }

    Item {
        id: gridContainer
        width: parent.width
        anchors.top: colorFilter.bottom
        anchors.topMargin: Theme.spacingS
        anchors.bottom: sortBar.top
        anchors.bottomMargin: 0

        ImageGrid {
            anchors.fill: parent
            model: tabHome.feedModel
            visible: !tabHome.loading
            favoritesList: tabHome.favoritesList
            currentWallpapers: tabHome.currentWallpapers
            onImageClicked: (modelData) => tabHome.imageClicked(modelData)
            onBlacklistClicked: (modelData) => tabHome.blacklistClicked(modelData)
            onDownloadClicked: (modelData) => tabHome.downloadClicked(modelData)
            onFavoriteClicked: (modelData) => tabHome.favoriteClicked(modelData)
        }

        GridView {
            anchors.fill: parent
            visible: tabHome.loading
            cellWidth: width / 3
            cellHeight: cellWidth
            interactive: false
            model: 9
            delegate: Item {
                width: GridView.view.cellWidth
                height: GridView.view.cellHeight
                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 2
                    color: Theme.surfaceContainerHighest
                    radius: 4
                    opacity: 0.5
                    SequentialAnimation on opacity {
                        loops: Animation.Infinite
                        running: tabHome.loading
                        NumberAnimation { from: 0.3; to: 0.7; duration: 800; easing.type: Easing.InOutQuad }
                        NumberAnimation { from: 0.7; to: 0.3; duration: 800; easing.type: Easing.InOutQuad }
                    }
                }
            }
        }

        StyledText {
            anchors.centerIn: parent
            text: "No images found"
            visible: !tabHome.loading && tabHome.feedModel && tabHome.feedModel.length === 0
            font.pixelSize: Theme.fontSizeLarge
            color: Theme.surfaceVariantText
        }
    }

    Item {
        id: sortBar
        width: parent.width
        height: 32
        anchors.bottom: currentWallpapersBar.top
        anchors.bottomMargin: Theme.spacingS

            ComboBox {
                id: sortSelector
                anchors.left: parent.left
                anchors.verticalCenter: parent.verticalCenter
                anchors.leftMargin: 2
                width: 140
                height: 32
                model: ["Smart", "Newest", "Oldest", "Source", "Unseen", "Random"]
                currentIndex: 0

                indicator: DankIcon {
                    x: sortSelector.width - width - sortSelector.rightPadding
                    y: sortSelector.topPadding + (sortSelector.availableHeight - height) / 2
                    name: "expand_more"
                    size: 16
                    color: Theme.surfaceVariantText
                }

                ToolTip.visible: hovered
                ToolTip.text: {
                    switch (currentText) {
                        case "Smart":
                            return "Ordena inteligentemente por relevancia y novedad.";
                        case "Newest":
                            return "Muestra los wallpapers más recientes primero.";
                        case "Oldest":
                            return "Muestra los wallpapers más antiguos primero.";
                        case "Source":
                            return "Agrupa los wallpapers por su origen (provider).";
                        case "Unseen":
                            return "Muestra solo wallpapers que no has visto.";
                        case "Random":
                            return "Ordena los wallpapers de forma aleatoria.";
                        default:
                            return "Seleccionar método de ordenación";
                    }
                }

                contentItem: Row {
                    anchors.verticalCenter: parent.verticalCenter
                    leftPadding: 10
                    spacing: 8
                    
                    DankIcon {
                        name: "sort"
                        size: 16
                        color: Theme.primary
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Text {
                        text: sortSelector.displayText
                        font: sortSelector.font
                        color: Theme.surfaceText
                        verticalAlignment: Text.AlignVCenter
                        elide: Text.ElideRight
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                background: Rectangle {
                    color: sortSelector.hovered ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
                    radius: 4
                    border.width: sortSelector.activeFocus ? 2 : 1
                    border.color: sortSelector.activeFocus ? Theme.primary : Theme.outline
                    Behavior on border.color { ColorAnimation { duration: 150 } }
                    Behavior on border.width { NumberAnimation { duration: 150 } }
                    Behavior on color { ColorAnimation { duration: 150 } }
                }

                delegate: ItemDelegate {
                    width: sortSelector.width
                    height: 32
                    contentItem: Text {
                        text: modelData
                        color: Theme.surfaceText
                        font: sortSelector.font
                        elide: Text.ElideRight
                        verticalAlignment: Text.AlignVCenter
                    }
                    background: Rectangle {
                        color: parent.highlighted ? Theme.surfaceContainerHigh : "transparent"
                    }
                    highlighted: sortSelector.highlightedIndex === index
                }

                popup: Popup {
                    y: sortSelector.height - 1
                    width: sortSelector.width
                    implicitHeight: contentItem.implicitHeight
                    padding: 1

                    contentItem: ListView {
                        clip: true
                        implicitHeight: contentHeight
                        model: sortSelector.popup.visible ? sortSelector.delegateModel : null
                        currentIndex: sortSelector.highlightedIndex
                        ScrollIndicator.vertical: ScrollIndicator { }
                    }

                    background: Rectangle {
                        color: Theme.surfaceContainerHighest
                        radius: 4
                        border.width: 1
                        border.color: Theme.surfaceContainerHigh
                    }
                }

                onActivated: {
                    Backend.feedSort = currentText.toLowerCase()
                    tabHome.pageChanged(1)
                }
            }

            TextField {
                id: pageInput
                anchors.centerIn: parent
                text: tabHome.feedPage
                width: 40
                height: 32
                horizontalAlignment: Text.AlignHCenter
                color: Theme.surfaceText
                background: Rectangle {
                    color: parent.activeFocus ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
                    radius: 4
                    border.width: parent.activeFocus ? 2 : 1
                    border.color: parent.activeFocus ? Theme.primary : Theme.outline
                    Behavior on border.color { ColorAnimation { duration: 150 } }
                    Behavior on border.width { NumberAnimation { duration: 150 } }
                }
                validator: IntValidator { bottom: 1 }
                onAccepted: {
                    var newPage = parseInt(text)
                    if (newPage !== tabHome.feedPage) {
                        tabHome.pageChanged(newPage)
                    }
                    focus = false
                }
                onFocusChanged: {
                    if (!focus && (text === "" || isNaN(parseInt(text)))) {
                        pageInput.text = tabHome.feedPage
                    }
                }
            }

            DankButton {
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                anchors.rightMargin: 2
                width: 32
                height: 32
                iconName: "undo"
                opacity: 0.5
                onClicked: Backend.undoWallpaper()
                ToolTip.visible: hovered
                ToolTip.text: "Deshacer último cambio"
            }
    }

    Item {
        id: currentWallpapersBar
        width: parent.width
        height: visible ? borderRect.height : 0
        anchors.bottom: parent.bottom
        visible: tabHome.displayedItems && tabHome.displayedItems.length > 0

        property int itemCount: tabHome.displayedItems ? tabHome.displayedItems.length : 0
        property int maxItemHeight: 100
        property int itemSize: {
            if (itemCount === 0) return 0;
            // Solo usamos maxItemHeight como base para la altura.
            // El ancho será dinámico según el aspect ratio.
            return maxItemHeight;
        }

        Rectangle {
            id: borderRect
            anchors.centerIn: parent
            width: wallpaperRow.implicitWidth + Theme.spacingM
            height: wallpaperRow.implicitHeight + Theme.spacingM
            color: "transparent"
            border.color: Theme.outline
            border.width: 1
            radius: 4

            Row {
                id: wallpaperRow
                anchors.centerIn: parent
                spacing: Theme.spacingS
                
                Repeater {
                    model: tabHome.displayedItems
                    delegate: Rectangle {
                        // Calculamos el ancho basado en el aspect ratio de la imagen si está lista
                        width: {
                            if (img.status === Image.Ready && img.sourceSize.height > 0) {
                                return (img.sourceSize.width / img.sourceSize.height) * currentWallpapersBar.itemSize
                            }
                            return currentWallpapersBar.itemSize // Fallback cuadrado
                        }
                        height: currentWallpapersBar.itemSize
                        radius: 4
                        color: Theme.surfaceContainerHighest
                        clip: true
                        enabled: !modelData._isDeleted
                        
                        Image {
                            id: img
                            anchors.fill: parent
                            source: modelData.thumbnail || ""
                            fillMode: Image.PreserveAspectFit
                            asynchronous: true
                            cache: false
                        }

                        Rectangle {
                            anchors.fill: parent
                            color: "#AA000000"
                            visible: modelData._isDeleted === true
                            z: 10
                            DankIcon {
                                anchors.centerIn: parent
                                name: "delete"
                                color: "white"
                                size: 24
                            }
                        }
                        
                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                            border.width: 1
                            border.color: Theme.primary
                            radius: 4
                        }

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: Backend.openImageExternally(modelData)
                        }

                        HoverHandler {
                            id: hoverHandler
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
                            radius: 0
                        }

                        Row {
                            anchors.bottom: parent.bottom
                            anchors.horizontalCenter: parent.horizontalCenter
                            anchors.bottomMargin: 8
                            spacing: 16
                            visible: hoverHandler.hovered

                            DankIcon {
                                name: "folder"
                                size: 20
                                color: "white"
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: Backend.openImageFolder(modelData) }
                            }
                            DankIcon {
                                name: "delete"
                                size: 20
                                color: "#FF5555"
                                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: { deleteDialog.modelData = modelData; deleteDialog.open() } }
                            }
                        }
                    }
                }
            }
        }
    }

    Popup {
        id: deleteDialog
        anchors.centerIn: parent
        width: 300
        padding: Theme.spacingM
        modal: true
        focus: true
        closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside

        background: Rectangle {
            color: Theme.surface
            radius: Theme.cornerRadius
            border.color: Theme.outline
            border.width: 1
        }

        property var modelData: null

        contentItem: ColumnLayout {
            spacing: Theme.spacingM

            StyledText {
                text: "Delete Wallpaper?"
                font.weight: Font.Bold
                Layout.alignment: Qt.AlignHCenter
            }

            Image {
                Layout.preferredWidth: 200
                Layout.preferredHeight: 112
                Layout.alignment: Qt.AlignHCenter
                source: deleteDialog.modelData ? deleteDialog.modelData.thumbnail : ""
                fillMode: Image.PreserveAspectCrop
                cache: false
            }

            StyledText {
                text: deleteDialog.modelData ? (deleteDialog.modelData.title || deleteDialog.modelData.id) : ""
                Layout.fillWidth: true
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.Wrap
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingM

                DankButton {
                    Layout.fillWidth: true
                    text: "Cancel"
                    onClicked: deleteDialog.close()
                }

                DankButton {
                    Layout.fillWidth: true
                    text: "Delete"
                    onClicked: {
                        if (deleteDialog.modelData) {
                            var items = []
                            if (tabHome.displayedItems) {
                                for (var i = 0; i < tabHome.displayedItems.length; i++) {
                                    items.push(tabHome.displayedItems[i])
                                }
                            }
                            for(var i=0; i<items.length; i++) {
                                if (items[i].id === deleteDialog.modelData.id) {
                                    var item = Object.assign({}, items[i])
                                    item._isDeleted = true
                                    items[i] = item
                                    break
                                }
                            }
                            tabHome.displayedItems = items
                            Backend.deleteWallpaper(deleteDialog.modelData.id)
                        }
                        deleteDialog.close()
                    }
                }
            }
        }
    }
}
