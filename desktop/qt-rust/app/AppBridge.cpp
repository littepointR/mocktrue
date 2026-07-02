#include "AppBridge.h"

#include <QFile>
#include <QFileInfo>
#include <QStringList>
#include <QTextStream>
#include <utility>

extern "C" {
const char *portweave_rust_smoke_message(void);
const char *portweave_rust_error_label(int kind);
int portweave_rust_error_kind_count(void);
}

namespace {
QString fromRustString(const char *value)
{
    return value ? QString::fromUtf8(value) : QStringLiteral("<null>");
}

QString localPathFromUrl(const QUrl &url)
{
    if (url.isLocalFile()) {
        return url.toLocalFile();
    }
    return url.toString(QUrl::PreferLocalFile);
}
} // namespace

AppBridge::AppBridge(QObject *parent)
    : QObject(parent)
    , m_currentView(QStringLiteral("serial"))
    , m_workspaceName(QStringLiteral("Untitled"))
    , m_statusMessage(QStringLiteral("Ready"))
{
    refreshSmoke();
}

void AppBridge::refreshSmoke()
{
    setSmokeMessage(fromRustString(portweave_rust_smoke_message()));
    setErrorKindSummary(buildErrorKindSummary());
}

void AppBridge::newWorkspace()
{
    setCurrentPath(QString());
    setWorkspaceName(QStringLiteral("Untitled"));
    replaceDocumentText(QString(), false);
    setCurrentView(QStringLiteral("serial"));
    setStatusMessage(QStringLiteral("New workspace created"));
}

bool AppBridge::openFile(const QUrl &fileUrl)
{
    const QString path = localPathFromUrl(fileUrl);
    if (path.isEmpty()) {
        setStatusMessage(QStringLiteral("Open cancelled: no file selected"));
        return false;
    }

    QFile file(path);
    if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
        setStatusMessage(QStringLiteral("Open failed: %1").arg(file.errorString()));
        return false;
    }

    const QByteArray raw = file.readAll();
    setCurrentPath(path);
    setWorkspaceName(QFileInfo(path).fileName());
    replaceDocumentText(QString::fromUtf8(raw), false);
    setCurrentView(QStringLiteral("serial"));
    setStatusMessage(QStringLiteral("Opened %1").arg(QFileInfo(path).fileName()));
    return true;
}

bool AppBridge::saveFile()
{
    if (m_currentPath.isEmpty()) {
        setStatusMessage(QStringLiteral("Save needs a path; use Save As"));
        return false;
    }
    return writeDocumentToPath(m_currentPath);
}

bool AppBridge::saveFileAs(const QUrl &fileUrl)
{
    const QString path = localPathFromUrl(fileUrl);
    if (path.isEmpty()) {
        setStatusMessage(QStringLiteral("Save As cancelled: no path selected"));
        return false;
    }
    setCurrentPath(path);
    setWorkspaceName(QFileInfo(path).fileName());
    return writeDocumentToPath(path);
}

void AppBridge::setCurrentView(QString value)
{
    if (value.isEmpty()) {
        value = QStringLiteral("serial");
    }
    if (m_currentView == value) {
        return;
    }
    m_currentView = std::move(value);
    emit currentViewChanged();
    setStatusMessage(QStringLiteral("View switched to %1").arg(m_currentView));
}

void AppBridge::setDocumentText(QString value)
{
    replaceDocumentText(std::move(value), true);
}

QString AppBridge::smokeMessage() const
{
    return m_smokeMessage;
}

QString AppBridge::errorKindSummary() const
{
    return m_errorKindSummary;
}

QString AppBridge::currentView() const
{
    return m_currentView;
}

QString AppBridge::currentPath() const
{
    return m_currentPath;
}

QString AppBridge::workspaceName() const
{
    return m_workspaceName;
}

QString AppBridge::documentText() const
{
    return m_documentText;
}

QString AppBridge::statusMessage() const
{
    return m_statusMessage;
}

bool AppBridge::isDirty() const
{
    return m_isDirty;
}

void AppBridge::setSmokeMessage(QString value)
{
    if (m_smokeMessage == value) {
        return;
    }
    m_smokeMessage = std::move(value);
    emit smokeMessageChanged();
}

void AppBridge::setErrorKindSummary(QString value)
{
    if (m_errorKindSummary == value) {
        return;
    }
    m_errorKindSummary = std::move(value);
    emit errorKindSummaryChanged();
}

void AppBridge::setCurrentPath(QString value)
{
    if (m_currentPath == value) {
        return;
    }
    m_currentPath = std::move(value);
    emit currentPathChanged();
}

void AppBridge::setWorkspaceName(QString value)
{
    if (value.isEmpty()) {
        value = QStringLiteral("Untitled");
    }
    if (m_workspaceName == value) {
        return;
    }
    m_workspaceName = std::move(value);
    emit workspaceNameChanged();
}

void AppBridge::setStatusMessage(QString value)
{
    if (m_statusMessage == value) {
        return;
    }
    m_statusMessage = std::move(value);
    emit statusMessageChanged();
}

void AppBridge::setDirty(bool value)
{
    if (m_isDirty == value) {
        return;
    }
    m_isDirty = value;
    emit isDirtyChanged();
}

void AppBridge::replaceDocumentText(QString value, bool dirty)
{
    if (m_documentText != value) {
        m_documentText = std::move(value);
        emit documentTextChanged();
    }
    setDirty(dirty);
    if (dirty) {
        setStatusMessage(QStringLiteral("Document edited"));
    }
}

QString AppBridge::buildErrorKindSummary() const
{
    const int count = portweave_rust_error_kind_count();
    QStringList labels;
    labels.reserve(count);
    for (int i = 0; i < count; ++i) {
        labels.append(fromRustString(portweave_rust_error_label(i)));
    }
    return labels.join(QStringLiteral(" | "));
}

bool AppBridge::writeDocumentToPath(const QString &path)
{
    QFile file(path);
    if (!file.open(QIODevice::WriteOnly | QIODevice::Text | QIODevice::Truncate)) {
        setStatusMessage(QStringLiteral("Save failed: %1").arg(file.errorString()));
        return false;
    }

    const QByteArray raw = m_documentText.toUtf8();
    if (file.write(raw) != raw.size()) {
        setStatusMessage(QStringLiteral("Save failed: %1").arg(file.errorString()));
        return false;
    }

    setDirty(false);
    setStatusMessage(QStringLiteral("Saved %1").arg(QFileInfo(path).fileName()));
    return true;
}
