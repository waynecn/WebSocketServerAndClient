#include "common.h"

#include <QProcess>

UserInfo g_stUserInfo;

QWebSocket g_WebSocket;

QString APPLICATION_DIR = "";

void RestartApp() {
    QString exeFile = APPLICATION_DIR + "/WebSocketChatClient.exe";
    qDebug() << "exeFile:" << exeFile;
    QProcess::startDetached(exeFile, QStringList());
}
