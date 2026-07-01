
#include <QCoreApplication>
#include <QDir>
#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QTimer>
#include <QUrl>

#include "AppBridge.h"
#include "WindowLifecycleProbe.h"

int main(int argc, char *argv[])
{
    QGuiApplication app(argc, argv);
    QCoreApplication::setApplicationName(QStringLiteral("PortWeaveQtRust"));
    QCoreApplication::setOrganizationName(QStringLiteral("PortWeave"));

    const bool smokeExit = QCoreApplication::arguments().contains(QStringLiteral("--smoke-exit"));

    AppBridge appBridge;
    WindowLifecycleProbe lifecycleProbe;

    QQmlApplicationEngine engine;
    engine.rootContext()->setContextProperty(QStringLiteral("appBridge"), &appBridge);
    engine.rootContext()->setContextProperty(QStringLiteral("lifecycleProbe"), &lifecycleProbe);

    const QString qmlPath = QDir(QCoreApplication::applicationDirPath()).filePath(QStringLiteral("qml/Main.qml"));
    engine.load(QUrl::fromLocalFile(qmlPath));
    if (engine.rootObjects().isEmpty()) {
        return -1;
    }

    if (smokeExit) {
        QTimer::singleShot(0, &app, &QCoreApplication::quit);
    }

    return app.exec();
}
