#include "mainwindow.h"
#include "logindialog.h"
#include "settingdlg.h"

#include <QApplication>
#include <QSettings>

int main(int argc, char *argv[])
{
    QApplication a(argc, argv);
    QCoreApplication::setOrganizationName("private");
    QCoreApplication::setOrganizationDomain("private.private.com");
    QCoreApplication::setApplicationName("websocketClient");

    QSettings settings;
    QString host = settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = settings.value(WEBSOCKET_SERVER_PORT).toString();
    if (host.isEmpty() || port.isEmpty()) {
        SettingDlg settingDlg;
        settingDlg.exec();
    }

    LoginDialog *loginDialog = new LoginDialog();
    int nRet = loginDialog->exec();
    MainWindow w;
    w.setWindowTitle(g_stUserInfo.strUserName);
    if (nRet == QDialog::Accepted) {
        w.show();
    } else {
        delete loginDialog;
        w.close();
        exit(-1);
    }
    delete loginDialog;
    return a.exec();
}
