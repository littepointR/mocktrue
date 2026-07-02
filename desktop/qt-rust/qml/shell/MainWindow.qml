import QtQuick
import QtQuick.Dialogs
import QtQuick.Window

Window {
    id: rootWindow
    objectName: "PortWeaveQtRustMainWindow"
    width: 1280
    height: 860
    visible: true
    title: "PortWeave Qt/Rust Skeleton - " + appBridge.workspaceName + (appBridge.isDirty ? " *" : "")
    color: "#111318"

    function handleToolbarAction(actionId) {
        if (actionId === "shell.smoke.new") {
            appBridge.newWorkspace()
        } else if (actionId === "shell.smoke.open") {
            openDialog.open()
        } else if (actionId === "shell.smoke.save") {
            if (appBridge.currentPath.length === 0) {
                saveAsDialog.open()
            } else {
                appBridge.saveFile()
            }
        } else if (actionId === "shell.smoke.save_as") {
            saveAsDialog.open()
        }
    }

    FileDialog {
        id: openDialog
        title: "Open workspace text file"
        fileMode: FileDialog.OpenFile
        nameFilters: ["Text files (*.txt *.md *.json *.csv *.log)", "All files (*)"]
        onAccepted: appBridge.openFile(selectedFile)
    }

    FileDialog {
        id: saveAsDialog
        title: "Save workspace text file"
        fileMode: FileDialog.SaveFile
        nameFilters: ["Text files (*.txt *.md *.json *.csv *.log)", "All files (*)"]
        onAccepted: appBridge.saveFileAs(selectedFile)
    }

    Rectangle {
        anchors.fill: parent
        color: "#111318"
    }

    MainToolbar {
        id: mainToolbar
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        height: 52
        onActionRequested: actionId => rootWindow.handleToolbarAction(actionId)
    }

    StatusBar {
        id: statusBar
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 32
    }

    Row {
        id: bodyRow
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: mainToolbar.bottom
        anchors.bottom: statusBar.top
        spacing: 0

        ActivityBar {
            id: activityBar
            width: 140
            height: parent.height
            onViewRequested: viewId => appBridge.setCurrentView(viewId)
        }

        Sidebar {
            id: sidebar
            width: 300
            height: parent.height
        }

        EditorGroups {
            id: editorGroups
            width: Math.max(0, parent.width - activityBar.width - sidebar.width)
            height: parent.height
        }
    }

    Component.onCompleted: {
        appBridge.refreshSmoke()
        lifecycleProbe.attachWindow(rootWindow)
    }
}
