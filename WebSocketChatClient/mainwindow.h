#ifndef MAINWINDOW_H
#define MAINWINDOW_H

#include "common.h"
#include "chatwidget.h"

#include <QMainWindow>
#include <QSplitter>
#include <QtWebSockets/QWebSocket>
#include <QUrl>
#include <QSettings>
#include <QKeyEvent>
#include <QTableWidgetItem>
#include <QTabWidget>

QT_BEGIN_NAMESPACE
namespace Ui { class MainWindow; }
QT_END_NAMESPACE

class MainWindow : public QMainWindow
{
    Q_OBJECT

public:
    MainWindow(QWidget *parent = nullptr);
    ~MainWindow();

private:
    void keyPressEvent(QKeyEvent *e);
    void keyReleaseEvent(QKeyEvent *e);

private slots:
    void OnWebSocketConnected();
    void OnWebSocketDisconnected();
    void OnWebSocketError(QAbstractSocket::SocketError err);
    void OnNewMessageArrived();

private:
    Ui::MainWindow      *ui;

    QString             m_strWsUrl;
    QSettings           m_Settings;
    bool                m_bCtrlPressed;

    ChatWidget          *m_pChatWidget;

signals:
    void webscketDisconnected();
    void websocketConnected();
};
#endif // MAINWINDOW_H
