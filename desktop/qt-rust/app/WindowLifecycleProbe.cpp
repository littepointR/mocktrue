
#include "WindowLifecycleProbe.h"

#include <QEvent>
#include <QScreen>
#include <QString>

WindowLifecycleProbe::WindowLifecycleProbe(QObject *parent)
    : QObject(parent)
{
}

void WindowLifecycleProbe::attachWindow(QWindow *window)
{
    if (!window) {
        appendEvent(QStringLiteral("attach skipped: no window"));
        return;
    }

    if (m_window == window) {
        appendEvent(QStringLiteral("attach skipped: same window"));
        return;
    }

    if (m_window) {
        m_window->removeEventFilter(this);
    }

    m_window = window;
    m_window->installEventFilter(this);

    QObject::connect(m_window, &QWindow::screenChanged, this, [this](QScreen *screen) {
        if (screen) {
            appendEvent(QStringLiteral("screen changed -> %1 (dpr=%2)")
                            .arg(screen->name())
                            .arg(screen->devicePixelRatio(), 0, 'f', 2));
        } else {
            appendEvent(QStringLiteral("screen changed -> <none>"));
        }
    });

    appendEvent(QStringLiteral("attached window -> %1x%2")
                    .arg(window->width())
                    .arg(window->height()));
}

void WindowLifecycleProbe::reset()
{
    m_eventLog.clear();
    emit eventLogChanged();
}

QStringList WindowLifecycleProbe::eventLog() const
{
    return m_eventLog;
}

QString WindowLifecycleProbe::lastEvent() const
{
    return m_eventLog.isEmpty() ? QStringLiteral("no lifecycle events yet") : m_eventLog.constLast();
}

bool WindowLifecycleProbe::eventFilter(QObject *watched, QEvent *event)
{
    if (watched != m_window) {
        return QObject::eventFilter(watched, event);
    }

    switch (event->type()) {
    case QEvent::Resize:
        appendEvent(QStringLiteral("resize -> %1x%2")
                        .arg(m_window->width())
                        .arg(m_window->height()));
        break;
    case QEvent::Move:
        appendEvent(QStringLiteral("move -> %1,%2")
                        .arg(m_window->x())
                        .arg(m_window->y()));
        break;
    case QEvent::WindowStateChange: {
        const auto state = m_window->windowState();
        if (state & Qt::WindowMinimized) {
            appendEvent(QStringLiteral("state -> minimized"));
        } else if (state & Qt::WindowMaximized) {
            appendEvent(QStringLiteral("state -> maximized"));
        } else if (state & Qt::WindowFullScreen) {
            appendEvent(QStringLiteral("state -> fullscreen"));
        } else {
            appendEvent(QStringLiteral("state -> restored"));
        }
        break;
    }
    case QEvent::DevicePixelRatioChange:
        appendEvent(QStringLiteral("dpi change -> dpr=%1").arg(m_window->devicePixelRatio(), 0, 'f', 2));
        break;
    case QEvent::Show:
        appendEvent(QStringLiteral("show -> visible"));
        break;
    case QEvent::Hide:
        appendEvent(QStringLiteral("hide -> hidden"));
        break;
    default:
        break;
    }

    return QObject::eventFilter(watched, event);
}

void WindowLifecycleProbe::appendEvent(QString message)
{
    m_eventLog.append(std::move(message));
    emit eventLogChanged();
}
