#include "mainwindow.h"
#include "logindialog.h"
#include "settingdlg.h"

#include <QApplication>
#include <QSettings>
#include <QCoreApplication>

int main(int argc, char *argv[])
{
    QApplication a(argc, argv);
    QCoreApplication::setOrganizationName("private");
    QCoreApplication::setOrganizationDomain("private.private.com");
    QCoreApplication::setApplicationName("websocketClient");

    APPLICATION_DIR = QCoreApplication::applicationDirPath();

    QSettings settings;
    QString host = settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = settings.value(WEBSOCKET_SERVER_PORT).toString();
    if (host.isEmpty() || port.isEmpty()) {
        SettingDlg *dlg = SettingDlg::GetInstance();
        dlg->exec();
    }

    LoginDialog *loginDialog = new LoginDialog();
    int nRet = loginDialog->exec();
    if (nRet == QDialog::Accepted) {
        APPLICATION_IMAGE_DIR = APPLICATION_DIR + "/images/" + g_stUserInfo.strUserId + "/";
    } else {
        APPLICATION_IMAGE_DIR = APPLICATION_DIR + "/images/";
    }
    MainWindow w;
    if (nRet == QDialog::Accepted) {
        w.setWindowTitle(g_stUserInfo.strUserName);
        w.show();
    } else {
        delete loginDialog;
        exit(-1);
    }
    delete loginDialog;
    return a.exec();
}
