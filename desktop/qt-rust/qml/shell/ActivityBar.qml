import QtQuick

Rectangle {
    id: root
    signal viewRequested(string viewId)

    color: "#151a25"
    border.color: "#2a3141"

    Column {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 12

        Text {
            text: "Activity"
            color: "#a8b1c7"
            font.pixelSize: 12
            font.capitalization: Font.AllUppercase
        }

        Repeater {
            model: [
                { "label": "Serial", "featureId": "serial" },
                { "label": "Settings", "featureId": "settings" }
            ]

            delegate: Rectangle {
                width: parent.width
                height: 48
                radius: 10
                color: appBridge.currentView === featureId ? "#365a9c" : (mouseArea.containsPress ? "#2b3b60" : mouseArea.containsMouse ? "#283145" : "#22293a")
                border.color: appBridge.currentView === featureId ? "#6ea1ff" : "#39445f"
                property string featureId: modelData.featureId

                Text {
                    anchors.centerIn: parent
                    text: modelData.label
                    color: "#eef1f7"
                    font.pixelSize: 15
                }

                MouseArea {
                    id: mouseArea
                    anchors.fill: parent
                    hoverEnabled: true
                    onClicked: root.viewRequested(featureId)
                }
            }
        }
    }
}
