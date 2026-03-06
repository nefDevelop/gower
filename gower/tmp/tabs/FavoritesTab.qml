import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Widgets
import qs.Common
import "../js"

Column {
    id: tabFavorites
    width: parent.width
    height: 500
    spacing: Theme.spacingS

    property var favoritesColors: []
    property string selectedColor: ""
    property var favoritesModel: null
    property var favoritesList: []

    signal loadFavorites()
    signal colorClicked(string color)
    signal imageClicked(var modelData)
    signal favoriteClicked(var modelData)

    ColorFilter {
        colors: tabFavorites.favoritesColors
        selectedColor: tabFavorites.selectedColor
        onColorClicked: (color) => tabFavorites.colorClicked(color)
    }

    ImageGrid {
        width: parent.width
        height: 500 - 48 - parent.spacing
        model: tabFavorites.favoritesModel
        favoritesList: tabFavorites.favoritesList
        onImageClicked: (modelData) => tabFavorites.imageClicked(modelData)
        onFavoriteClicked: (modelData) => tabFavorites.favoriteClicked(modelData)
    }
}
