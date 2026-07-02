import QtQuick

Rectangle {
    id: root
    signal actionRequested(string actionId)

    color: "#1f2533"
    border.color: "#2f374a"
    radius: 0

    Row {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 10

        Text {
            text: appBridge.workspaceName + (appBridge.isDirty ? " •" : "")
            color: "#f3f4f8"
            font.pixelSize: 18
            font.bold: true
            verticalAlignment: Text.AlignVCenter
            width: 280
            elide: Text.ElideRight
        }

        Repeater {
            model: [
                { "label": "New", "featureId": "shell.smoke.new" },
                { "label": "Open", "featureId": "shell.smoke.open" },
                { "label": "Save", "featureId": "shell.smoke.save" },
                { "label": "Save As", "featureId": "shell.smoke.save_as" }
            ]

            delegate: Rectangle {
                width: 96
                height: 28
                radius: 8
                color: mouseArea.containsPress ? "#4d7cff" : mouseArea.containsMouse ? "#3b455c" : "#30384a"
                border.color: "#4b5a77"
                property string featureId: modelData.featureId

                Text {
                    anchors.centerIn: parent
                    text: modelData.label
                    color: "#ffffff"
                    font.pixelSize: 13
                }

                MouseArea {
                    id: mouseArea
                    anchors.fill: parent
                    hoverEnabled: true
                    onClicked: root.actionRequested(featureId)
                }
            }
        }

        Text {
            text: appBridge.currentPath.length > 0 ? appBridge.currentPath : "No file selected"
            color: "#a8b1c7"
            font.pixelSize: 12
            verticalAlignment: Text.AlignVCenter
            width: Math.max(120, parent.width - 706)
            elide: Text.ElideMiddle
        }
    }
}
