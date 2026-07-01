#pragma once

#include <QObject>
#include <QString>
#include <QUrl>

class AppBridge final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString smokeMessage READ smokeMessage NOTIFY smokeMessageChanged)
    Q_PROPERTY(QString errorKindSummary READ errorKindSummary NOTIFY errorKindSummaryChanged)
    Q_PROPERTY(QString currentView READ currentView WRITE setCurrentView NOTIFY currentViewChanged)
    Q_PROPERTY(QString currentPath READ currentPath NOTIFY currentPathChanged)
    Q_PROPERTY(QString workspaceName READ workspaceName NOTIFY workspaceNameChanged)
    Q_PROPERTY(QString documentText READ documentText WRITE setDocumentText NOTIFY documentTextChanged)
    Q_PROPERTY(QString statusMessage READ statusMessage NOTIFY statusMessageChanged)
    Q_PROPERTY(bool isDirty READ isDirty NOTIFY isDirtyChanged)

public:
    explicit AppBridge(QObject *parent = nullptr);

    Q_INVOKABLE void refreshSmoke();
    Q_INVOKABLE void newWorkspace();
    Q_INVOKABLE bool openFile(const QUrl &fileUrl);
    Q_INVOKABLE bool saveFile();
    Q_INVOKABLE bool saveFileAs(const QUrl &fileUrl);
    Q_INVOKABLE void setCurrentView(QString value);

    void setDocumentText(QString value);

    [[nodiscard]] QString smokeMessage() const;
    [[nodiscard]] QString errorKindSummary() const;
    [[nodiscard]] QString currentView() const;
    [[nodiscard]] QString currentPath() const;
    [[nodiscard]] QString workspaceName() const;
    [[nodiscard]] QString documentText() const;
    [[nodiscard]] QString statusMessage() const;
    [[nodiscard]] bool isDirty() const;

signals:
    void smokeMessageChanged();
    void errorKindSummaryChanged();
    void currentViewChanged();
    void currentPathChanged();
    void workspaceNameChanged();
    void documentTextChanged();
    void statusMessageChanged();
    void isDirtyChanged();

private:
    void setSmokeMessage(QString value);
    void setErrorKindSummary(QString value);
    void setCurrentPath(QString value);
    void setWorkspaceName(QString value);
    void setStatusMessage(QString value);
    void setDirty(bool value);
    void replaceDocumentText(QString value, bool dirty);
    [[nodiscard]] QString buildErrorKindSummary() const;
    [[nodiscard]] bool writeDocumentToPath(const QString &path);

    QString m_smokeMessage;
    QString m_errorKindSummary;
    QString m_currentView;
    QString m_currentPath;
    QString m_workspaceName;
    QString m_documentText;
    QString m_statusMessage;
    bool m_isDirty = false;
};
