import QtQuick

Rectangle {
    id: root
    color: "#1f2533"
    border.color: "#2f374a"

    Row {
        anchors.fill: parent
        anchors.margins: 10
        spacing: 16

        Text {
            text: "Status: " + appBridge.statusMessage
            color: "#f2f4fa"
            font.pixelSize: 12
            elide: Text.ElideRight
            width: parent.width * 0.48
        }

        Text {
            text: "View: " + appBridge.currentView + " | " + (appBridge.isDirty ? "Dirty" : "Clean")
            color: appBridge.isDirty ? "#f3b74f" : "#a8b1c7"
            font.pixelSize: 12
            width: parent.width * 0.22
            elide: Text.ElideRight
        }

        Text {
            text: lifecycleProbe.lastEvent
            color: "#a8b1c7"
            font.pixelSize: 12
            horizontalAlignment: Text.AlignRight
            width: parent.width * 0.28
            elide: Text.ElideRight
        }
    }
}
