import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Widgets
import qs.Common
import "../js"

Column {
    id: searchTab
    width: parent.width
    height: 500
    spacing: Theme.spacingS

    property bool isSearching: false
    property var searchModel: null
    property var providerList: []
    property var customProviderList: []
    property bool feedLoading: false
    property bool searchLoading: false
    property int appsPage: 1
    property int appsLimit: 9
    property var plugins: []
    property var activePlugin: null
    property var favoritesList: []

    signal search(string text, string provider)
    signal appClicked(var modelData)
    signal imageClicked(var modelData)
    signal blacklistClicked(var modelData)
    signal downloadClicked(var modelData)
    signal favoriteClicked(var modelData)
    signal pageChanged(int newPage)

    Row {
        width: parent.width
        spacing: Theme.spacingS

        FloatingTextField {
            id: searchInput
            width: (parent.width - parent.spacing) / 2
            height: 40
            label: "Search..."
            onAccepted: {
                if (text === "") {
                    searchTab.isSearching = false
                    return
                }
                searchTab.isSearching = true
                searchTab.search(text, appSelector.currentText)
            }
        }

        ComboBox {
            id: appSelector
            width: (parent.width - parent.spacing) / 2
            height: 40
            model: {
                var names = []
                for(var i=0; i<searchTab.providerList.length; i++) names.push(searchTab.providerList[i].name)
                for(var i=0; i<searchTab.customProviderList.length; i++) names.push(searchTab.customProviderList[i].name)
                return names
            }

            indicator: DankIcon {
                x: appSelector.width - width - appSelector.rightPadding
                y: appSelector.topPadding + (appSelector.availableHeight - height) / 2
                name: "expand_more"
                size: 16
                color: Theme.surfaceVariantText
            }

            contentItem: Row {
                anchors.verticalCenter: parent.verticalCenter
                leftPadding: 10
                spacing: 8
                
                DankIcon {
                    name: "apps"
                    size: 16
                    color: Theme.primary
                    anchors.verticalCenter: parent.verticalCenter
                }

                Text {
                    text: appSelector.displayText
                    font: appSelector.font
                    color: Theme.surfaceText
                    verticalAlignment: Text.AlignVCenter
                    elide: Text.ElideRight
                    anchors.verticalCenter: parent.verticalCenter
                }
            }

            background: Rectangle {
                color: appSelector.hovered ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
                radius: 4
                border.width: appSelector.activeFocus ? 2 : 1
                border.color: appSelector.activeFocus ? Theme.primary : Theme.outline
                Behavior on border.color { ColorAnimation { duration: 150 } }
                Behavior on border.width { NumberAnimation { duration: 150 } }
                Behavior on color { ColorAnimation { duration: 150 } }
            }

            delegate: ItemDelegate {
                width: appSelector.width
                height: 32
                contentItem: Text {
                    text: modelData
                    color: Theme.surfaceText
                    font: appSelector.font
                    elide: Text.ElideRight
                    verticalAlignment: Text.AlignVCenter
                }
                background: Rectangle {
                    color: parent.highlighted ? Theme.surfaceContainerHigh : "transparent"
                }
                highlighted: appSelector.highlightedIndex === index
            }

            popup: Popup {
                y: appSelector.height - 1
                width: appSelector.width
                implicitHeight: contentItem.implicitHeight
                padding: 1

                contentItem: ListView {
                    clip: true
                    implicitHeight: contentHeight
                    model: appSelector.popup.visible ? appSelector.delegateModel : null
                    currentIndex: appSelector.highlightedIndex
                    ScrollIndicator.vertical: ScrollIndicator { }
                }

                background: Rectangle {
                    color: Theme.surfaceContainerHighest
                    radius: 4
                    border.width: 1
                    border.color: Theme.surfaceContainerHigh
                }
            }
        }
    }

    Item {
        width: parent.width
        height: parent.height - 36 - (paginationRow.visible ? 32 + parent.spacing : 0) - (parent.spacing * 2)
        
        GridView {
            anchors.fill: parent
            visible: searchTab.feedLoading || searchTab.searchLoading
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
                        running: searchTab.feedLoading || searchTab.searchLoading
                        NumberAnimation { from: 0.3; to: 0.7; duration: 800; easing.type: Easing.InOutQuad }
                        NumberAnimation { from: 0.7; to: 0.3; duration: 800; easing.type: Easing.InOutQuad }
                    }
                }
            }
        }

        GridView {
            anchors.fill: parent
            visible: !searchTab.feedLoading && !searchTab.isSearching
            clip: true
            cellWidth: width / 3
            cellHeight: cellWidth
            model: {
                var start = (searchTab.appsPage - 1) * searchTab.appsLimit
                return searchTab.plugins.slice(start, start + searchTab.appsLimit)
            }
            delegate: Item {
                width: GridView.view.cellWidth - 4
                height: GridView.view.cellHeight - 4
                
                Rectangle {
                    anchors.fill: parent
                    color: "transparent"
                    border.color: Theme.outline
                    border.width: 1
                    radius: 4

                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onClicked: searchTab.appClicked(modelData)
                    }
                }
            }
        }

        ImageGrid {
            anchors.fill: parent
            visible: !searchTab.searchLoading && searchTab.isSearching
            model: searchTab.searchModel
            favoritesList: searchTab.favoritesList
            onImageClicked: (modelData) => searchTab.imageClicked(modelData)
            onBlacklistClicked: (modelData) => searchTab.blacklistClicked(modelData)
            onDownloadClicked: (modelData) => searchTab.downloadClicked(modelData)
            onFavoriteClicked: (modelData) => searchTab.favoriteClicked(modelData)
        }

        StyledText {
            anchors.centerIn: parent
            text: "No results found"
            visible: !searchTab.searchLoading && searchTab.isSearching && searchModel.length === 0
            font.pixelSize: Theme.fontSizeLarge
            color: Theme.surfaceVariantText
        }
    }

    RowLayout {
        id: paginationRow
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: Theme.spacingM
        visible: !searchTab.isSearching && searchTab.plugins.length > searchTab.appsLimit

        TextField {
            id: pageInput
            text: searchTab.appsPage
            Layout.alignment: Qt.AlignVCenter
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
                var maxPages = Math.max(1, Math.ceil(searchTab.plugins.length / searchTab.appsLimit))
                if (newPage > maxPages) newPage = maxPages
                
                if (newPage !== searchTab.appsPage) {
                    searchTab.pageChanged(newPage)
                }
                
                pageInput.text = newPage
                focus = false
            }
        }
    }
}
