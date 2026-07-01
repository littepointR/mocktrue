
#pragma once

#include <QObject>
#include <QPointer>
#include <QStringList>
#include <QWindow>

class WindowLifecycleProbe final : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QStringList eventLog READ eventLog NOTIFY eventLogChanged)
    Q_PROPERTY(QString lastEvent READ lastEvent NOTIFY eventLogChanged)

public:
    explicit WindowLifecycleProbe(QObject *parent = nullptr);

    Q_INVOKABLE void attachWindow(QWindow *window);
    Q_INVOKABLE void reset();

    [[nodiscard]] QStringList eventLog() const;
    [[nodiscard]] QString lastEvent() const;

signals:
    void eventLogChanged();

protected:
    bool eventFilter(QObject *watched, QEvent *event) override;

private:
    void appendEvent(QString message);

    QPointer<QWindow> m_window;
    QStringList m_eventLog;
};
