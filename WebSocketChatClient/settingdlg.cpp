#include "settingdlg.h"
#include "ui_settingdlg.h"

#include <QMessageBox>
#include <QSettings>

SettingDlg *SettingDlg::m_pInstance = nullptr;
SettingDlg::SettingDlg(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::SettingDlg)
{
    Qt::WindowFlags flags= this->windowFlags();
    setWindowFlags(flags&~Qt::WindowContextHelpButtonHint);
    ui->setupUi(this);

    QSettings setting;
    QString host = setting.value(WEBSOCKET_SERVER_HOST, "").toString();
    QString port = setting.value(WEBSOCKET_SERVER_PORT, "").toString();
    if (!host.isEmpty()) {
        ui->ipLineEdit->setText(host);
    }
    if (!port.isEmpty()) {
        ui->portLineEdit->setText(port);
    }
}

SettingDlg::~SettingDlg()
{
    delete ui;
}

void SettingDlg::on_okBtn_clicked()
{
    QString ip = ui->ipLineEdit->text();
    QString port = ui->portLineEdit->text();
    if (ip.isEmpty() || port.isEmpty()) {
        QString msg = "IP或端口不能为空";
        QMessageBox box;
        box.setWindowTitle("提示");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        return;
    }

    QStringList ipList = ip.split(".");
    if (ipList.size() != 4) {
        QString msg = "IP地址填写有误";
        QMessageBox box;
        box.setWindowTitle("提示");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        return;
    }
    for (int i = 0; i < ipList.size(); ++i) {
        int nItem = ipList[i].toInt();
        if (nItem < 0 || nItem > 255) {
            QString msg = "IP地址非法";
            QMessageBox box;
            box.setWindowTitle("提示");
            box.setText(msg);
            box.addButton("确定", QMessageBox::AcceptRole);
            box.exec();
            return;
        }
    }

    bool bOk = false;
    int nPort = port.toInt(&bOk);
    if (!bOk || nPort < 0) {
        QString msg = "端口非法";
        QMessageBox box;
        box.setWindowTitle("提示");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        emit restartApp();
        return;
    }

    QSettings setting;

    bool bExit = false;
    QString oldIP = setting.value(WEBSOCKET_SERVER_HOST).toString();
    QString oldPort = setting.value(WEBSOCKET_SERVER_PORT).toString();
    if ((!oldIP.isEmpty() || !oldPort.isEmpty()) && (oldIP != ip || oldPort != port)) {
        QString msg = "配置发生改变,需要重启程序";
        QMessageBox box;
        box.setWindowTitle("提示");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        bExit = true;
    }

    setting.setValue(WEBSOCKET_SERVER_HOST, ip);
    setting.setValue(WEBSOCKET_SERVER_PORT, port);
    accept();
    if (bExit) {
        emit restartApp();
        exit(-1);
    }
}

void SettingDlg::on_cancelBtn_clicked()
{
    reject();
}
