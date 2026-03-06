import QtQuick
import QtQuick.Controls
import QtQuick.Controls.Basic // Para usar Overlay
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Services
import qs.Widgets
import QtQuick.Layouts
import qs.Modules.Plugins
import QtQuick.Window
import "js"
import "tabs"

// Usamos PluginComponent: la base estándar para plugins de DMS
PluginComponent {
    id: root

    property int currentTab: 0
    property var feedColors: []
    property var favoritesColors: []
    property string userHome: ""
    property bool updateFinished: false
    property int feedPage: 1
    property int feedLimit: 9
    property var feedList: []
    property bool feedLoading: false
    property bool popoutCooldown: false
    property var favoritesList: []
    property var favoritesModel: []
    property bool refreshing: false
    property int interval: 30
    property int batteryThreshold: 20
    property bool autoDownload: true
    property bool respectDarkMode: true
    property bool pauseOnLowBattery: true
    property string wallpaperPath: ""
    property bool indexWallpapers: false
    property var providerList: []
    property var plugins: []
    property var customProviderList: []
    property string homeSelectedColor: ""
    property string favoritesSelectedColor: ""
    property var activePlugin: null
    property int appsPage: 1
    property int appsLimit: 9
    property bool isSearching: false
    property bool searchLoading: false
    property var feedModel: []
    property var searchModel: []
    property string pendingWallpaperId: ""
    property string monitorMode: "Clone"
    property var monitors: []
    property var currentWallpapers: []
    property var currentWallpaperItems: []
    property bool fromFavorites: false



    Timer {
        id: cooldownTimer
        interval: 1500
        onTriggered: root.popoutCooldown = false
    }

    Component.onCompleted: {
        Backend.initialize()
    }

    Connections {
        target: Backend
        function onUserHomeChanged() {
            root.userHome = Backend.userHome
            root.wallpaperPath = Backend.userHome + "/.gower/wallpapers"
        }
        function onMonitorsChanged() {
            root.monitors = Backend.monitors
        }
        function onCurrentWallpapersChanged() {
            console.info("Main: Current wallpapers updated: " + JSON.stringify(Backend.currentWallpapers))
            root.currentWallpapers = Backend.currentWallpapers
        }
        function onCurrentWallpaperItemsChanged() {
            root.currentWallpaperItems = Backend.currentWallpaperItems
        }
        function onColorsChanged(feedColors, favoritesColors) {
            root.feedColors = feedColors
            root.favoritesColors = favoritesColors
        }
        function onFeedChanged(feed) {
            root.feedLoading = false
            root.feedModel = feed
            if (feed.length === 0) {
                console.warn("Feed is empty. Check gower configuration or network.")
            }
        }
        function onFavoritesChanged(favorites) {
            root.favoritesModel = favorites
            root.favoritesList = favorites.map(function(item) { return item.id })
        }
        function onSearchChanged(searchResult) {
            root.searchModel = searchResult
        }
        function onFeedNeedsReload() {
            root.feedLoading = true
            Backend.loadFeed(root.homeSelectedColor)
        }
        function onFavoritesNeedReload() {
            Backend.loadFavorites(root.favoritesSelectedColor)
        }
        function onConfigChanged() {
            var config = Backend.config
            root.wallpaperPath = config.paths.wallpapers
            root.indexWallpapers = config.paths.index_wallpapers !== undefined ? config.paths.index_wallpapers : false
            root.interval = config.behavior.change_interval
            root.autoDownload = config.behavior.auto_download
            root.respectDarkMode = config.behavior.respect_dark_mode
            root.pauseOnLowBattery = config.power.pause_on_low_battery
            root.batteryThreshold = config.power.low_battery_threshold
            root.fromFavorites = config.behavior.from_favorites !== undefined ? config.behavior.from_favorites : false

            if (config.behavior && config.behavior.multi_monitor) {
                var mode = config.behavior.multi_monitor
                root.monitorMode = mode.charAt(0).toUpperCase() + mode.slice(1)
            }

            var p = config.providers                        
            var list = []
            if (p && typeof p === 'object') {
                for (var key in p) {
                    if (p.hasOwnProperty(key)) {
                        list.push({
                            key: key,
                            name: p[key].name || (key.charAt(0).toUpperCase() + key.slice(1)),
                            enabled: p[key].enabled,
                            apiKey: p[key].api_key,
                            hasApiKey: p[key].hasOwnProperty("api_key"),
                            searchUrl: p[key].search_url || ""
                        });
                    }
                }
            }
            root.providerList = list;

            var gp = config.generic_providers
            var customList = []
            if (gp && typeof gp === 'object') {
                for (var key in gp) {
                    if (gp.hasOwnProperty(key)) {
                        customList.push({
                            key: key,
                            name: gp[key].name || "",
                            enabled: gp[key].enabled,
                            apiKey: gp[key].api_key || "",
                            searchUrl: gp[key].search_url || "",
                            hasApiKey: true,
                            isCustom: true
                        });
                    }
                }
            }
            root.customProviderList = customList
        }
        function onUpdateFinished() {
            root.updateFinished = true
        }
    }

    // Define cómo se ve el widget en la barra
    horizontalBarPill: Component {
        DankIcon {
            name: "photo" // Nombre del icono (Material Symbols)
            size: Theme.iconSize - 6
            color: Theme.surfaceText // Usa el color del tema automáticamente
            anchors.verticalCenter: parent.verticalCenter
        }
    }

    // Define el contenido del menú flotante (Popout)
    popoutContent: Component {
        PopoutComponent {
            headerText: root.activePlugin ? root.activePlugin.name : "Gower Options"

            onVisibleChanged: {
                if (visible) {
                    Backend.loadCurrentWallpapers();
                    Backend.checkAndLoadColors();
                    if (root.feedModel.length === 0 && !root.feedLoading) {
                        root.feedLoading = true;
                        Backend.loadFeed(root.homeSelectedColor);
                    }
                }
            }

            Item {
                width: parent.width
                height: root.activePlugin ? pluginColumn.implicitHeight : mainColumn.implicitHeight

                Popup {
                    id: monitorSelectorPopup
                    anchors.centerIn: parent
                    width: 300
                    modal: true
                    focus: true
                    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside
                    padding: Theme.spacingM

                    background: Rectangle {
                        color: Theme.surface
                        radius: Theme.cornerRadius
                        border.color: Theme.outline
                        border.width: 1
                    }

                    contentItem: Column {
                        spacing: Theme.spacingS

                        StyledText {
                            text: "Select Monitor"
                            font.weight: Font.Bold
                            color: Theme.onSurface
                            anchors.horizontalCenter: parent.horizontalCenter
                            bottomPadding: Theme.spacingS
                        }

                        DankButton {
                            width: parent.width
                            height: 36
                            text: "All Monitors"
                            iconName: "wallpaper"
                            onClicked: {
                                Backend.setWallpaper(root.pendingWallpaperId)
                                monitorSelectorPopup.close()
                            }
                        }

                        Repeater {
                            model: root.monitors
                            delegate: DankButton {
                                width: parent.width
                                height: 36
                                text: modelData.Name + (modelData.Primary ? " (Primary)" : "")
                                iconName: "monitor"
                                onClicked: {
                                    Backend.setWallpaper(root.pendingWallpaperId, modelData.Name)
                                    monitorSelectorPopup.close()
                                }
                            }
                        }

                        DankButton {
                            width: parent.width
                            height: 36
                            text: "Cancel"
                            iconName: "close"
                            onClicked: monitorSelectorPopup.close()
                        }
                    }
                }

                Column {
                    id: mainColumn
                    width: parent.width
                    visible: root.activePlugin === null
                    spacing: Theme.spacingM

                // Barra de navegación (Pestañas)
                Row {
                    width: parent.width
                    spacing: Theme.spacingXS
                    
                    Repeater {
                        model: [
                            { icon: "home", label: "Inicio" },
                            { icon: "search", label: "Apps" },
                            { icon: "star", label: "Favoritos" },
                            { icon: "settings", label: "Ajustes" }
                        ]

                        Rectangle {
                            width: (parent.width - (parent.spacing * 3)) / 4
                            height: 36
                            radius: Theme.cornerRadius
                            color: root.currentTab === index ? Theme.primary : Theme.surfaceContainerHigh

                            DankIcon {
                                anchors.centerIn: parent
                                name: modelData.icon
                                size: 20
                                color: root.currentTab === index ? "#ffffff" : Theme.surfaceText
                            }

                            MouseArea {
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: {
                                    root.currentTab = index;
                                    if (index === 0) {
                                        Backend.loadCurrentWallpapers()
                                        root.feedLoading = true
                                        Backend.loadFeed(root.homeSelectedColor)
                                    }
                                    if (index === 2) Backend.loadFavorites(root.favoritesSelectedColor)
                                }
                            }
                        }
                    }
                }

                // Contenedor de contenido
                StackLayout {
                    width: parent.width
                    currentIndex: root.currentTab
                    height: implicitHeight
                    Behavior on height { NumberAnimation { duration: 200; easing.type: Easing.OutQuad } }

                    Item {
                        width: parent.width
                        implicitHeight: homeTab.implicitHeight
                        HomeTab {
                        id: homeTab
                        width: parent.width
                        feedColors: root.feedColors
                        selectedColor: root.homeSelectedColor
                        feedPage: root.feedPage
                        feedModel: root.feedModel
                        favoritesList: root.favoritesList
                        currentWallpapers: root.currentWallpapers
                        currentWallpaperItems: root.currentWallpaperItems
                        feedLimit: root.feedLimit
                        loading: root.feedLoading
                        onLoadFeed: {
                            root.feedLoading = true
                            Backend.loadFeed(root.homeSelectedColor)
                        }
                        onColorClicked: (color) => {
                            if (root.homeSelectedColor === color) {
                                root.homeSelectedColor = ""
                            } else {
                                root.homeSelectedColor = color
                            }
                            root.feedPage = 1
                            root.feedLoading = true
                            Backend.loadFeed(root.homeSelectedColor)
                        }
                        onImageClicked: (modelData) => {
                            if (root.monitorMode === "Distinct") {
                                root.pendingWallpaperId = modelData.id
                                monitorSelectorPopup.open()
                            } else {
                                Backend.setWallpaper(modelData.id)
                            }
                        }
                        onBlacklistClicked: (modelData) => Backend.blacklist(modelData.id)
                        onDownloadClicked: (modelData) => Backend.download(modelData.id)
                        onFavoriteClicked: (modelData) => {
                            var isFav = root.favoritesList.indexOf(modelData.id) !== -1
                            if (isFav) {
                                Backend.removeFavorite(modelData.id)
                            } else {
                                Backend.addFavorite(modelData.id)
                            }
                        }
                        onPageChanged: (newPage) => {
                            root.feedPage = newPage
                            Backend.feedPage = newPage
                            root.feedLoading = true
                            Backend.loadFeed(root.homeSelectedColor)
                        }
                        }

                        MouseArea {
                            anchors.fill: parent
                            acceptedButtons: Qt.NoButton
                            onWheel: (wheel) => {
                                if (root.feedLoading) return
                                if (wheel.angleDelta.y < 0 && root.feedModel.length === root.feedLimit) {
                                    root.feedPage += 1
                                    Backend.feedPage = root.feedPage
                                    root.feedLoading = true
                                    Backend.loadFeed(root.homeSelectedColor)
                                } else if (wheel.angleDelta.y > 0 && root.feedPage > 1) {
                                    root.feedPage -= 1
                                    Backend.feedPage = root.feedPage
                                    root.feedLoading = true
                                    Backend.loadFeed(root.homeSelectedColor)
                                }
                            }
                        }
                    }

                    Item {
                        width: parent.width
                        implicitHeight: searchTab.implicitHeight
                        SearchTab {
                        id: searchTab
                        width: parent.width
                        isSearching: root.isSearching
                        searchModel: root.searchModel
                        providerList: root.providerList
                        customProviderList: root.customProviderList
                        feedLoading: root.feedLoading
                        searchLoading: root.searchLoading
                        appsPage: root.appsPage
                        appsLimit: root.appsLimit
                        plugins: root.plugins
                        activePlugin: root.activePlugin
                        favoritesList: root.favoritesList
                        onSearch: (text, provider) => Backend.search(text, provider)
                        onAppClicked: (modelData) => root.activePlugin = modelData
                        onImageClicked: (modelData) => {
                            if (root.monitorMode === "Distinct") {
                                root.pendingWallpaperId = modelData.id
                                monitorSelectorPopup.open()
                            } else {
                                Backend.setWallpaper(modelData.id)
                            }
                        }
                        onBlacklistClicked: (modelData) => Backend.blacklist(modelData.id)
                        onDownloadClicked: (modelData) => Backend.download(modelData.id)
                        onFavoriteClicked: (modelData) => {
                            var isFav = root.favoritesList.indexOf(modelData.id) !== -1
                            if (isFav) {
                                Backend.removeFavorite(modelData.id)
                            } else {
                                Backend.addFavorite(modelData.id)
                            }
                        }
                        onPageChanged: (newPage) => root.appsPage = newPage
                        }

                        MouseArea {
                            anchors.fill: parent
                            acceptedButtons: Qt.NoButton
                            onWheel: (wheel) => {
                                var maxPages = Math.ceil(root.searchModel.length / root.appsLimit)
                                if (wheel.angleDelta.y < 0 && root.appsPage < maxPages) {
                                    root.appsPage += 1
                                } else if (wheel.angleDelta.y > 0 && root.appsPage > 1) {
                                    root.appsPage -= 1
                                }
                            }
                        }
                    }

                    FavoritesTab {
                        favoritesColors: root.favoritesColors
                        selectedColor: root.favoritesSelectedColor
                        favoritesModel: root.favoritesModel
                        favoritesList: root.favoritesList
                        onLoadFavorites: Backend.loadFavorites(root.favoritesSelectedColor)
                        onColorClicked: (color) => {
                            if (root.favoritesSelectedColor === color) {
                                root.favoritesSelectedColor = ""
                            } else {
                                root.favoritesSelectedColor = color
                            }
                            Backend.loadFavorites(root.favoritesSelectedColor)
                        }
                        onImageClicked: (modelData) => {
                            if (root.monitorMode === "Distinct") {
                                root.pendingWallpaperId = modelData.id
                                monitorSelectorPopup.open()
                            } else {
                                Backend.setWallpaper(modelData.id)
                            }
                        }
                        onFavoriteClicked: (modelData) => Backend.removeFavorite(modelData.id)
                    }

                    SettingsTab {
                        wallpaperPath: root.wallpaperPath
                        indexWallpapers: root.indexWallpapers
                        interval: root.interval
                        autoDownload: root.autoDownload
                        respectDarkMode: root.respectDarkMode
                        pauseOnLowBattery: root.pauseOnLowBattery
                        batteryThreshold: root.batteryThreshold
                        isLaptop: Backend.isLaptop
                        monitorMode: root.monitorMode
                        monitorCount: root.monitors.length
                        daemonRunning: Backend.daemonRunning
                        fromFavorites: root.fromFavorites
                        providerList: root.providerList
                        customProviderList: root.customProviderList
                        onOpenFolderPicker: {
                            Backend.openFolderPicker(function(path) {
                                root.wallpaperPath = path
                                Backend.setConfig("paths.wallpapers", path)
                            })
                        }
                        onOpenProviders: {
                            Backend.loadConfig()
                        }
                        onConfigSet: (key, value) => Backend.setConfig(key, value)
                        onRefreshMonitors: Backend.loadMonitors()
                        onToggleDaemon: (enable) => Backend.toggleDaemon(enable)
                        onRefreshDaemon: Backend.checkDaemonStatus()
                    }
                }
            }
        }
    }
}

}