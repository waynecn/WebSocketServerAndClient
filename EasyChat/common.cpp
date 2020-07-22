#include "common.h"

#include <QProcess>

UserInfo g_stUserInfo;

QWebSocket g_WebSocket;

QString APPLICATION_DIR = "";

QString APPLICATION_IMAGE_DIR = "";

void RestartApp() {
    QString exeFile = APPLICATION_DIR + "/EasyChat.exe";
    qDebug() << "exeFile:" << exeFile;
    QProcess::startDetached(exeFile, QStringList());
}
