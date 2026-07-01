import QtQuick

Rectangle {
    id: root
    color: "#0f131c"
    border.color: "#2a3141"

    Item {
        anchors.fill: parent
        anchors.margins: 16

        Column {
            id: serialView
            anchors.fill: parent
            spacing: 12
            visible: appBridge.currentView === "serial"

            Text {
                text: "Serial workspace document"
                color: "#f7f8fb"
                font.pixelSize: 24
                font.bold: true
            }

            Text {
                text: appBridge.currentPath.length > 0 ? appBridge.currentPath : "New unsaved workspace"
                color: "#b3bed6"
                font.pixelSize: 13
                elide: Text.ElideMiddle
                width: parent.width
            }

            Rectangle {
                width: parent.width
                height: Math.max(320, parent.height - 92)
                radius: 10
                color: "#111827"
                border.color: appBridge.isDirty ? "#f3b74f" : "#30384a"
                clip: true

                Flickable {
                    id: editorFlickable
                    anchors.fill: parent
                    anchors.margins: 12
                    contentWidth: width
                    contentHeight: Math.max(editor.contentHeight, height)
                    clip: true

                    TextEdit {
                        id: editor
                        width: editorFlickable.width
                        height: Math.max(editor.contentHeight, editorFlickable.height)
                        text: appBridge.documentText
                        color: "#eef1f7"
                        selectionColor: "#4d7cff"
                        selectedTextColor: "#ffffff"
                        font.family: "Menlo"
                        font.pixelSize: 13
                        wrapMode: TextEdit.Wrap
                        textFormat: TextEdit.PlainText
                        focus: true
                        onTextChanged: {
                            if (text !== appBridge.documentText) {
                                appBridge.documentText = text
                            }
                        }
                    }
                }
            }
        }

        Column {
            id: settingsView
            anchors.fill: parent
            spacing: 12
            visible: appBridge.currentView === "settings"

            Text {
                text: "Settings"
                color: "#f7f8fb"
                font.pixelSize: 24
                font.bold: true
            }

            Text {
                text: "Rust smoke result: " + appBridge.smokeMessage
                color: "#dce2f2"
                font.pixelSize: 16
                wrapMode: Text.WordWrap
                width: parent.width
            }

            Text {
                text: "Rust classified error labels: " + appBridge.errorKindSummary
                color: "#b3bed6"
                font.pixelSize: 13
                wrapMode: Text.WordWrap
                width: parent.width
            }

            Rectangle {
                width: parent.width
                height: 1
                color: "#2d374a"
            }

            Text {
                text: "Window lifecycle log"
                color: "#f0f3fa"
                font.pixelSize: 14
                font.bold: true
            }

            ListView {
                id: lifecycleList
                width: parent.width
                height: Math.max(180, parent.height - 180)
                clip: true
                model: lifecycleProbe.eventLog
                spacing: 4

                delegate: Text {
                    width: lifecycleList.width
                    text: "• " + modelData
                    color: "#cbd3e8"
                    font.pixelSize: 12
                    wrapMode: Text.WordWrap
                }
            }
        }
    }
}
