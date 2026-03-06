import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Widgets
import qs.Common
import "../js"

Item {
    id: settingsTab
    
    property string wallpaperPath: ""
    property bool indexWallpapers: false
    property int interval: 30
    property bool autoDownload: true
    property bool respectDarkMode: true
    property bool pauseOnLowBattery: true
    property int batteryThreshold: 20
    property string monitorMode: "Clone"
    property bool isLaptop: false
    property int monitorCount: 0
    property bool daemonRunning: false
    property bool fromFavorites: false
    property var providerList: []
    property var customProviderList: []

    signal openFolderPicker()
    signal openProviders()
    signal configSet(string key, var value)
    signal refreshMonitors()
    signal toggleDaemon(bool enable)
    signal refreshDaemon()

    component SettingsSwitch : Switch {
        id: control
        scale: 0.8
        
        indicator: Rectangle {
            implicitWidth: 48
            implicitHeight: 26
            x: control.leftPadding
            y: parent.height / 2 - height / 2
            radius: 13
            color: control.checked ? Theme.primary : Theme.surfaceContainerHighest
            border.color: control.checked ? Theme.primary : Theme.outline
            border.width: 1

            Behavior on color { ColorAnimation { duration: 150 } }
            Behavior on border.color { ColorAnimation { duration: 150 } }

            Rectangle {
                x: control.checked ? parent.width - width - 2 : 2
                width: 22
                height: 22
                radius: 11
                anchors.verticalCenter: parent.verticalCenter
                color: control.checked ? Theme.onPrimary : Theme.surfaceVariantText
                Behavior on x { NumberAnimation { duration: 150; easing.type: Easing.OutQuad } }
                Behavior on color { ColorAnimation { duration: 150 } }
            }
        }
    }

    component SortComboBox : ComboBox {
        id: control
        model: ["new", "hot", "top", "controversial", "mix"]
        
        indicator: DankIcon {
            x: control.width - width - control.rightPadding
            y: control.topPadding + (control.availableHeight - height) / 2
            name: "expand_more"
            size: 16
            color: Theme.surfaceVariantText
        }

        contentItem: Text {
            leftPadding: 10
            text: control.displayText
            font: control.font
            color: Theme.surfaceText
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
        }

        background: Rectangle {
            color: control.hovered ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
            radius: 4
            border.width: control.activeFocus ? 2 : 1
            border.color: control.activeFocus ? Theme.primary : Theme.outline
        }

        delegate: ItemDelegate {
            width: control.width
            height: 32
            contentItem: Text {
                text: modelData
                color: Theme.surfaceText
                font: control.font
                elide: Text.ElideRight
                verticalAlignment: Text.AlignVCenter
            }
            background: Rectangle {
                color: parent.highlighted ? Theme.surfaceContainerHigh : "transparent"
            }
            highlighted: control.highlightedIndex === index
        }

        popup: Popup {
            y: control.height - 1
            width: control.width
            implicitHeight: contentItem.implicitHeight
            padding: 1
            contentItem: ListView {
                clip: true
                implicitHeight: contentHeight
                model: control.popup.visible ? control.delegateModel : null
                currentIndex: control.highlightedIndex
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

    StackLayout {
        id: stack
        anchors.fill: parent
        currentIndex: 0

        Item {
            ScrollView {
                anchors.fill: parent
                anchors.bottomMargin: 30 // Espacio para la versión
                clip: true
                contentWidth: width

                Column {
                    width: parent.width
                    spacing: Theme.spacingM
                    bottomPadding: Theme.spacingL

                    // --- DAEMON ---
                    Column {
                        width: parent.width
                        spacing: Theme.spacingS
                        
                        StyledText { 
                            text: "Daemon"
                            font.weight: Font.Bold
                            color: Theme.primary
                        }

                        RowLayout {
                            width: parent.width
                            spacing: Theme.spacingS
                            
                            SettingsSwitch { 
                                checked: settingsTab.daemonRunning
                                onClicked: {
                                    settingsTab.toggleDaemon(checked)
                                    settingsTab.configSet("behavior.daemon_enabled", checked)
                                }
                                ToolTip.visible: hovered
                                ToolTip.text: "Activar/Desactivar el servicio en segundo plano"
                            }
                            StyledText { 
                                text: settingsTab.daemonRunning ? "Running" : "Stopped"
                                color: settingsTab.daemonRunning ? Theme.primary : Theme.surfaceVariantText
                                Layout.alignment: Qt.AlignVCenter 
                            }

                            Item { Layout.fillWidth: true }

                            StyledText { 
                                text: "Interval (min):"
                                Layout.alignment: Qt.AlignVCenter 
                            }
                            TextField {
                                Layout.preferredWidth: 60
                                Layout.preferredHeight: 32
                                text: settingsTab.interval
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
                                onEditingFinished: settingsTab.configSet("behavior.change_interval", parseInt(text) || 30)
                            }
                        }

                        Row {
                            spacing: Theme.spacingS
                            SettingsSwitch { checked: settingsTab.autoDownload; onClicked: { settingsTab.autoDownload = checked; settingsTab.configSet("behavior.auto_download", checked) } }
                            StyledText { text: "Auto Download"; anchors.verticalCenter: parent.verticalCenter }
                        }
                        
                        Row {
                            spacing: Theme.spacingS
                            SettingsSwitch { checked: settingsTab.respectDarkMode; onClicked: { settingsTab.respectDarkMode = checked; settingsTab.configSet("behavior.respect_dark_mode", checked) } }
                            StyledText { text: "Respect Dark Mode"; anchors.verticalCenter: parent.verticalCenter }
                        }

                        Row {
                            spacing: Theme.spacingS
                            SettingsSwitch {
                                checked: settingsTab.fromFavorites
                                onClicked: { settingsTab.fromFavorites = checked; settingsTab.configSet("behavior.from_favorites", checked) }
                                ToolTip.visible: hovered
                                ToolTip.text: "Si está activo, solo usará wallpapers de tus favoritos"
                            }
                            StyledText { text: "Usar solo favoritos"; anchors.verticalCenter: parent.verticalCenter }
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: Theme.surfaceContainerHighest }

                    // --- GENERAL ---
                    Column {
                        width: parent.width
                        spacing: Theme.spacingS
                        
                        StyledText { text: "General"; font.weight: Font.Bold; color: Theme.primary }
                        
                        Row {
                            width: parent.width
                            spacing: Theme.spacingS
                            TextField {
                                width: parent.width - 40 - parent.spacing; height: 32
                                text: settingsTab.wallpaperPath
                                color: Theme.surfaceText
                                background: Rectangle {
                                    color: parent.activeFocus ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
                                    radius: 4
                                    border.width: parent.activeFocus ? 2 : 1
                                    border.color: parent.activeFocus ? Theme.primary : Theme.outline
                                    Behavior on border.color { ColorAnimation { duration: 150 } }
                                    Behavior on border.width { NumberAnimation { duration: 150 } }
                                }
                                readOnly: true
                            }
                            DankButton {
                                width: 40; height: 32; iconName: "folder_open"
                                onClicked: settingsTab.openFolderPicker()
                            }
                        }

                        Row {
                            spacing: Theme.spacingS
                            SettingsSwitch { 
                                checked: settingsTab.indexWallpapers
                                onClicked: {
                                    settingsTab.configSet("paths.index_wallpapers", checked)
                                }
                            }
                            StyledText { text: "Indexar carpeta seleccionada"; anchors.verticalCenter: parent.verticalCenter }
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: Theme.surfaceContainerHighest }

                    // --- PROVIDERS ---
                    Column {
                        width: parent.width
                        spacing: Theme.spacingS
                        
                        StyledText { 
                            text: "Providers"
                            font.weight: Font.Bold
                            color: Theme.primary
                        }

                        Rectangle {
                            width: parent.width
                            height: 36
                            color: Theme.surfaceContainerHighest
                            radius: 4
                            
                            StyledText {
                                text: "Configure Providers"
                                anchors.centerIn: parent
                                font.weight: Font.Bold
                            }
                            
                            MouseArea {
                                anchors.fill: parent
                                cursorShape: Qt.PointingHandCursor
                                onClicked: {
                                    settingsTab.openProviders()
                                    stack.currentIndex = 1
                                }
                            }
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: Theme.surfaceContainerHighest }

                    // --- DISPLAY ---
                    Column {
                        width: parent.width
                        spacing: Theme.spacingS

                        StyledText {
                            text: "Display"
                            font.weight: Font.Bold
                            color: Theme.primary
                        }

                        Row {
                            width: parent.width
                            spacing: Theme.spacingS
                            StyledText { text: "Mode:"; anchors.verticalCenter: parent.verticalCenter }
                            ComboBox {
                                id: monitorModeComboBox
                                width: 120
                                height: 32
                                model: ["Clone", "Distinct"]
                                currentIndex: settingsTab.monitorMode === "Distinct" ? 1 : 0
                                onActivated: {
                                    settingsTab.configSet("behavior.multi_monitor", monitorModeComboBox.currentText.toLowerCase())
                                }

                                indicator: DankIcon {
                                    x: monitorModeComboBox.width - width - monitorModeComboBox.rightPadding
                                    y: monitorModeComboBox.topPadding + (monitorModeComboBox.availableHeight - height) / 2
                                    name: "expand_more"
                                    size: 16
                                    color: Theme.surfaceVariantText
                                }

                                contentItem: Row {
                                    anchors.verticalCenter: parent.verticalCenter
                                    leftPadding: 10
                                    spacing: 8
                                    
                                    DankIcon {
                                        name: "desktop_windows"
                                        size: 16
                                        color: Theme.primary
                                        anchors.verticalCenter: parent.verticalCenter
                                    }

                                    Text {
                                        text: monitorModeComboBox.displayText
                                        font: monitorModeComboBox.font
                                        color: Theme.surfaceText
                                        verticalAlignment: Text.AlignVCenter
                                        elide: Text.ElideRight
                                        anchors.verticalCenter: parent.verticalCenter
                                    }
                                }

                                background: Rectangle {
                                    color: monitorModeComboBox.hovered ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
                                    radius: 4
                                    border.width: monitorModeComboBox.activeFocus ? 2 : 1
                                    border.color: monitorModeComboBox.activeFocus ? Theme.primary : Theme.outline
                                    Behavior on border.color { ColorAnimation { duration: 150 } }
                                    Behavior on border.width { NumberAnimation { duration: 150 } }
                                    Behavior on color { ColorAnimation { duration: 150 } }
                                }

                                delegate: ItemDelegate {
                                    width: monitorModeComboBox.width
                                    height: 32
                                    contentItem: Text {
                                        text: modelData
                                        color: Theme.surfaceText
                                        font: monitorModeComboBox.font
                                        elide: Text.ElideRight
                                        verticalAlignment: Text.AlignVCenter
                                    }
                                    background: Rectangle {
                                        color: parent.highlighted ? Theme.surfaceContainerHigh : "transparent"
                                    }
                                    highlighted: monitorModeComboBox.highlightedIndex === index
                                }

                                popup: Popup {
                                    y: monitorModeComboBox.height - 1
                                    width: monitorModeComboBox.width
                                    implicitHeight: contentItem.implicitHeight
                                    padding: 1

                                    contentItem: ListView {
                                        clip: true
                                        implicitHeight: contentHeight
                                        model: monitorModeComboBox.popup.visible ? monitorModeComboBox.delegateModel : null
                                        currentIndex: monitorModeComboBox.highlightedIndex
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
                            StyledText {
                                text: "(" + settingsTab.monitorCount + " Monitores)"
                                anchors.verticalCenter: parent.verticalCenter
                                color: Theme.surfaceContainerHighest
                                visible: settingsTab.monitorCount > 0
                            }
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: Theme.surfaceContainerHighest }

                    // --- POWER ---
                    Column {
                        width: parent.width
                        spacing: Theme.spacingS
                        visible: settingsTab.isLaptop
                        
                        StyledText { 
                            text: "Power"
                            font.weight: Font.Bold
                            color: Theme.primary
                        }

                        Row {
                            spacing: Theme.spacingS
                            SettingsSwitch { checked: settingsTab.pauseOnLowBattery; onClicked: { settingsTab.pauseOnLowBattery = checked; settingsTab.configSet("power.pause_on_low_battery", checked) } }
                            StyledText { text: "Pause on Low Battery"; anchors.verticalCenter: parent.verticalCenter }
                        }
                        
                        Row {
                            width: parent.width
                            spacing: Theme.spacingS
                            StyledText { text: "Threshold (%):"; anchors.verticalCenter: parent.verticalCenter }
                            TextField {
                                width: 60; height: 32
                                text: settingsTab.batteryThreshold
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
                                onEditingFinished: settingsTab.configSet("power.low_battery_threshold", parseInt(text) || 20)
                            }
                        }
                    }
                }
            }
            
            StyledText {
                text: "Versión 0.0.5"
                color: Theme.surfaceVariantText
                font.pixelSize: Theme.fontSizeSmall
                anchors.bottom: parent.bottom
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.bottomMargin: 5
            }
        }

        // Providers View
        Column {
            id: providersWindowLayout
            width: parent.width
            height: parent.height
            spacing: Theme.spacingM
            property alias addProviderDialog: addProviderDialog

            Item { // Header
                id: titleItem
                width: parent.width
                height: 32

                StyledText {
                    text: "Providers"
                    font.weight: Font.Bold
                    font.pixelSize: Theme.fontSizeLarge
                    anchors.centerIn: parent
                }

                DankButton {
                    anchors.left: parent.left
                    anchors.verticalCenter: parent.verticalCenter
                    width: 32
                    height: 32
                    iconName: "arrow_back"
                    onClicked: stack.currentIndex = 0
                }
            }

            ScrollView { // Scrollable area for provider lists
                width: parent.width
                height: providersWindowLayout.height - titleItem.height - addProviderButton.height - (providersWindowLayout.spacing * 2)
                clip: true
                contentHeight: providersColumn.implicitHeight

                Column {
                    id: providersColumn
                    width: parent.width
                    spacing: Theme.spacingM

                    // --- Built-in Providers ---
                    Repeater {
                        model: settingsTab.providerList
                        delegate: Column {
                            width: parent.width
                            spacing: Theme.spacingS

                            RowLayout {
                                width: parent.width
                                spacing: Theme.spacingM
                                StyledText {
                                    text: modelData.name
                                    font.weight: Font.Bold
                                    Layout.alignment: Qt.AlignVCenter
                                }
                                
                                Item {
                                    visible: !modelData.hasApiKey && modelData.key !== "reddit"
                                    Layout.fillWidth: true
                                }

                                SortComboBox {
                                    visible: modelData.key === "reddit"
                                    Layout.fillWidth: true
                                    Layout.preferredHeight: 40
                                    currentIndex: model.indexOf(modelData.sort || "hot")
                                    onActivated: Backend.setConfig("providers." + modelData.key + ".sort", currentText)
                                }

                                FloatingTextField {
                                    visible: modelData.hasApiKey
                                    Layout.fillWidth: true
                                    Layout.preferredHeight: 40
                                    label: "API Key"
                                    text: modelData.apiKey || ""
                                    echoMode: TextInput.Password
                                    onEditingFinished: Backend.setConfig("providers." + modelData.key + ".api_key", text)
                                }
                                SettingsSwitch {
                                    checked: modelData.enabled
                                    Layout.alignment: Qt.AlignVCenter
                                    onClicked: Backend.setConfig("providers." + modelData.key + ".enabled", checked)
                                }
                            }
                            Rectangle {
                                width: parent.width; height: 1;
                                color: Theme.surfaceContainerHighest
                                visible: index < settingsTab.providerList.length - 1
                            }
                        }
                    }

                    // --- Separator ---
                    Rectangle {
                        width: parent.width; height: 1
                        color: Theme.outline
                        visible: settingsTab.customProviderList.length > 0 && settingsTab.providerList.length > 0
                    }

                    // --- Custom Providers ---
                    Repeater {
                        model: settingsTab.customProviderList
                        delegate: Column {
                            width: parent.width
                            spacing: Theme.spacingS

                            RowLayout {
                                width: parent.width
                                spacing: Theme.spacingM

                                RowLayout {
                                    Layout.fillWidth: true
                                    spacing: Theme.spacingS
                                    StyledText {
                                        text: (modelData.sort !== undefined ? "r/" : "") + modelData.name
                                        font.weight: Font.Bold
                                    }
                                    Item { Layout.fillWidth: true }
                                    SortComboBox {
                                        visible: modelData.sort !== undefined
                                        Layout.preferredWidth: 110
                                        Layout.preferredHeight: 32
                                        currentIndex: model.indexOf(modelData.sort || "hot")
                                        onActivated: Backend.setConfig("generic_providers." + modelData.name + ".sort", currentText)
                                    }
                                }

                                DankButton {
                                    iconName: "delete"
                                    width: 32; height: 32
                                    onClicked: Backend.removeProvider(modelData.name)
                                }
                                SettingsSwitch {
                                    checked: modelData.enabled
                                    Layout.alignment: Qt.AlignVCenter
                                    onClicked: Backend.setConfig("generic_providers." + modelData.name + ".enabled", checked)
                                }
                            }

                            Rectangle {
                                width: parent.width; height: 1;
                                color: Theme.surfaceContainerHighest
                                visible: index < settingsTab.customProviderList.length - 1
                            }
                        }
                    }
                }
            }

            DankButton {
                id: addProviderButton
                width: parent.width
                height: 40
                text: "Add Provider"
                iconName: "add"
                onClicked: {
                    newProviderName.text = ""
                    newProviderUrl.text = ""
                    newProviderKey.text = ""
                    addProviderDialog.visible = true
                }
            }
        }
    }

    Rectangle {
        id: addProviderDialog
        anchors.fill: parent
        color: "#AA000000"
        visible: false
        z: 100

        MouseArea { anchors.fill: parent; onClicked: addProviderDialog.visible = false }

        Rectangle {
            width: parent.width - 64
            height: dialogColumn.implicitHeight + (Theme.spacingM * 2)
            anchors.centerIn: parent
            color: Theme.surfaceContainer
            radius: Theme.cornerRadius
            border.width: 1
            border.color: Theme.outline

            Column {
                id: dialogColumn
                anchors.fill: parent
                anchors.margins: Theme.spacingM
                spacing: Theme.spacingM

                StyledText {
                    text: "Add Custom Provider"
                    font.weight: Font.Bold
                    anchors.horizontalCenter: parent.horizontalCenter
                }

                FloatingTextField {
                    id: newProviderName
                    width: parent.width
                    label: "Name (e.g. wallpapers)"
                }

                FloatingTextField {
                    id: newProviderUrl
                    width: parent.width
                    property bool isReddit: text.trim().toLowerCase().startsWith("r/")
                    label: isReddit ? "Reddit Community (Detected)" : "URL (e.g. https://api...)"
                }

                Column {
                    width: parent.width
                    spacing: 4
                    visible: newProviderUrl.isReddit

                    StyledText {
                        text: "Sort By"
                        font.pixelSize: 12
                        color: Theme.surfaceVariantText
                    }

                    ComboBox {
                        id: redditSortSelector
                        width: parent.width
                        height: 40
                        model: ["new", "hot", "top", "controversial", "mix"]
                        currentIndex: 0

                        indicator: DankIcon {
                            x: parent.width - width - parent.rightPadding
                            y: parent.topPadding + (parent.availableHeight - height) / 2
                            name: "expand_more"
                            size: 16
                            color: Theme.surfaceVariantText
                        }

                        contentItem: Text {
                            leftPadding: 10
                            text: redditSortSelector.displayText
                            font: redditSortSelector.font
                            color: Theme.surfaceText
                            verticalAlignment: Text.AlignVCenter
                            elide: Text.ElideRight
                        }

                        background: Rectangle {
                            color: redditSortSelector.hovered ? Theme.surfaceContainerHigh : Theme.surfaceContainerHighest
                            radius: 4
                            border.width: redditSortSelector.activeFocus ? 2 : 1
                            border.color: redditSortSelector.activeFocus ? Theme.primary : Theme.outline
                        }

                        delegate: ItemDelegate {
                            width: redditSortSelector.width
                            height: 32
                            contentItem: Text {
                                text: modelData
                                color: Theme.surfaceText
                                font: redditSortSelector.font
                                elide: Text.ElideRight
                                verticalAlignment: Text.AlignVCenter
                            }
                            background: Rectangle {
                                color: parent.highlighted ? Theme.surfaceContainerHigh : "transparent"
                            }
                            highlighted: redditSortSelector.highlightedIndex === index
                        }

                        popup: Popup {
                            y: redditSortSelector.height - 1
                            width: redditSortSelector.width
                            implicitHeight: contentItem.implicitHeight
                            padding: 1
                            contentItem: ListView {
                                clip: true
                                implicitHeight: contentHeight
                                model: redditSortSelector.popup.visible ? redditSortSelector.delegateModel : null
                                currentIndex: redditSortSelector.highlightedIndex
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

                FloatingTextField {
                    id: newProviderKey
                    width: parent.width
                    label: "API Key (Optional)"
                    visible: !newProviderUrl.isReddit
                }

                RowLayout {
                    width: parent.width
                    spacing: Theme.spacingM
                    DankButton {
                        Layout.fillWidth: true
                        text: "Cancel"
                        onClicked: addProviderDialog.visible = false
                    }
                    DankButton {
                        Layout.fillWidth: true
                        text: "Add"
                        onClicked: {
                            if (newProviderUrl.isReddit) {
                                var channel = newProviderUrl.text.trim()
                                if (channel.toLowerCase().startsWith("r/")) {
                                    channel = channel.substring(2)
                                }
                                Backend.addRedditProvider(channel, redditSortSelector.currentText)
                            } else {
                                Backend.addProvider(newProviderName.text, newProviderUrl.text, newProviderKey.text)
                            }
                            addProviderDialog.visible = false
                        }
                    }
                }
            }
        }
    }
}
