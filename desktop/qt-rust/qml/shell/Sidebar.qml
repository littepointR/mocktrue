import QtQuick

Rectangle {
    id: root
    color: "#171c27"
    border.color: "#2a3141"

    Column {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 10

        Text {
            text: "Workspace"
            color: "#a8b1c7"
            font.pixelSize: 12
            font.capitalization: Font.AllUppercase
        }

        Text {
            text: appBridge.workspaceName
            color: "#f2f4fa"
            font.pixelSize: 18
            font.bold: true
            width: parent.width
            elide: Text.ElideRight
        }

        Text {
            text: appBridge.isDirty ? "Unsaved changes" : "Saved"
            color: appBridge.isDirty ? "#f3b74f" : "#72d572"
            font.pixelSize: 13
        }

        Text {
            text: appBridge.currentPath.length > 0 ? appBridge.currentPath : "No file path yet"
            color: "#c7cee0"
            font.pixelSize: 12
            wrapMode: Text.WrapAnywhere
            width: parent.width
        }

        Rectangle {
            width: parent.width
            height: 1
            color: "#30384a"
        }

        Text {
            text: "Active view: " + appBridge.currentView
            color: "#f2f4fa"
            font.pixelSize: 14
        }

        Text {
            text: "Status: " + appBridge.statusMessage
            color: "#9aa5bf"
            font.pixelSize: 12
            wrapMode: Text.WordWrap
            width: parent.width
        }

        Rectangle {
            width: parent.width
            height: 1
            color: "#30384a"
        }

        Text {
            text: "Rust smoke: " + appBridge.smokeMessage
            color: "#c7cee0"
            font.pixelSize: 12
            wrapMode: Text.WordWrap
            width: parent.width
        }
    }
}
