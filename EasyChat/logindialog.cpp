#include "logindialog.h"
#include "ui_logindialog.h"
#include "common.h"
#include "registerdlg.h"
#include "settingdlg.h"

#include <QTextCodec>
#include <QNetworkReply>
#include <QJsonParseError>
#include <QJsonDocument>
#include <QMessageBox>
#include <QJsonObject>
#include <QCloseEvent>
#include <QSettings>

LoginDialog::LoginDialog(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::LoginDialog)
{
    Qt::WindowFlags flags= this->windowFlags();
    setWindowFlags(flags&~Qt::WindowContextHelpButtonHint);
    ui->setupUi(this);

    QSettings settings;
    QString userName = settings.value(WEBSOCKET_USER_NAME, "").toString();
    ui->userNameEdit->setText(userName);
    QString pwd = settings.value(WEBSOCKET_USER_PWD, "").toString();
    bool bRemberPwd = settings.value(WEBSOCKET_REMBER_PWD, false).toBool();
    if (bRemberPwd) {
        ui->remberPwdCheckBox->setChecked(bRemberPwd);
        ui->passwordEdit->setText(pwd);
    }

    m_pAccessManager = new QNetworkAccessManager();
    connect(m_pAccessManager, SIGNAL(finished(QNetworkReply *)), this, SLOT(replyFinished(QNetworkReply *)));
}

LoginDialog::~LoginDialog()
{
    delete ui;
}

void LoginDialog::closeEvent(QCloseEvent *e) {
    reject();
    close();
    e->accept();
}

void LoginDialog::keyPressEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = true;
    }
    if (m_bCtrlPressed && e->key() == Qt::Key_E) {
        SettingDlg *dlg = SettingDlg::GetInstance();
        dlg->exec();
    }
    if (e->key() == Qt::Key_Return || e->key() == Qt::Key_Enter) {
        on_loginBtn_clicked();
    }
}

void LoginDialog::keyReleaseEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = false;
    }
}

void LoginDialog::on_loginBtn_clicked()
{
    m_eRequestAction = REQUEST_LOGIN;

    QString userName = ui->userNameEdit->text();
    QString password = ui->passwordEdit->text();
    if (userName.isEmpty() || password.isEmpty()) {
        QString msg = "用户名或密码不能为空";
        QMessageBox box;
        box.setWindowTitle("警告");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        return;
    }
    QSettings settings;
    QString host = settings.value(CURRENT_SERVER_HOST, "").toString();
    if (host.isEmpty()) {
        settings.value(WEBSOCKET_SERVER_HOST, "").toString();
    }
    QString port = settings.value(WEBSOCKET_SERVER_PORT).toString();
    QUrl url(QString("http://%1:%2/loginnew").arg(host).arg(port));
    qDebug() << "url:" << url;
    QNetworkRequest req(url);
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    QTextCodec *codec = QTextCodec::codecForName("utf-8");
    //QString loginInfo = "{\"username\":\"" + userName + "\", \"password\":\"" + password + "\",\"clientversion\":\"" + APPLICATION_VERSION + "\"}";
    //QString loginInfo = "{\"username\":\"" + userName + "\", \"password\":\"" + password + "\",\"clientversion\":\"" + "1.0.12" + "\"}";
    QString loginInfo = "{\"username\":\"" + userName + "\", \"password\":\"" + password + "\",\"clientversion\":\"1.0.12\"}";
    QByteArray bData = codec->fromUnicode(loginInfo);

    m_pAccessManager->post(req, bData);
    ui->loginBtn->setEnabled(false);
}

void LoginDialog::on_registerBtn_clicked()
{
    //this->hide();
    RegisterDlg *dlg = new RegisterDlg();
    dlg->exec();

    dlg->close();
    delete dlg;
    //this->show();
}

void LoginDialog::replyFinished(QNetworkReply *reply) {
    m_sNewClientFileName = "";
    int statusCode = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    if(reply->error() == QNetworkReply::NoError && statusCode == 200)
    {
        QByteArray baData = reply->readAll();
        //将波形数据从waveData中抽取出来，只保留浮点数
        QJsonParseError jsonErr;
        QJsonDocument jsonDoc = QJsonDocument::fromJson(baData, &jsonErr);
        if (jsonErr.error != QJsonParseError::NoError) {
            QString msg = "解析rsponse数据发生错误";
            qDebug() << msg;
            QMessageBox box;
            box.setWindowTitle("警告");
            box.setText(msg);
            box.addButton("确定", QMessageBox::AcceptRole);
            box.exec();
            reply->deleteLater();
            return;
        }

        qDebug() << "jsonDoc:" << jsonDoc;
        Q_ASSERT(jsonDoc.isObject());
        bool bRet = jsonDoc["Success"].toBool();
        if (m_eRequestAction == REQUEST_LOGIN) {
            if (!bRet) {
                QString msg = "登陆失败：" + jsonDoc["Msg"].toString();
                qDebug() << msg;
                QMessageBox box;
                box.setWindowTitle("提示");
                box.setText(msg);
                box.addButton("确定", QMessageBox::AcceptRole);
                box.exec();
                ui->loginBtn->setEnabled(true);
                return;
            }
            g_stUserInfo.strUserName = jsonDoc["Username"].toString();
            g_stUserInfo.strPwd = ui->passwordEdit->text();
            g_stUserInfo.strUserId = QString::number(jsonDoc["Id"].toInt());
            g_stUserInfo.strLoginTime = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
            if (jsonDoc["NewClient"].toObject()["Flag"].toBool()) {
                QMessageBox box;
                box.setWindowTitle("提示");
                box.setText("发现新版本客户端，是否更新");
                box.addButton("确定", QMessageBox::AcceptRole);
                box.addButton("取消", QMessageBox::RejectRole);
                m_sNewClientFileName = jsonDoc["NewClient"].toObject()["FileName"].toString();
                connect(&box, SIGNAL(accepted()), this, SLOT(downLoadNewClient()));
                box.exec();
            }

            QSettings settings;
            settings.setValue(WEBSOCKET_USER_NAME, ui->userNameEdit->text());
            if (ui->remberPwdCheckBox->isChecked()) {
                settings.setValue(WEBSOCKET_REMBER_PWD, true);
                settings.setValue(WEBSOCKET_USER_PWD, ui->passwordEdit->text());
            } else {
                settings.setValue(WEBSOCKET_REMBER_PWD, false);
            }

            accept();
        }
    }
}

void LoginDialog::downLoadNewClient() {
    QSettings settings;
    QString host = settings.value(CURRENT_SERVER_HOST, "").toString();
    if (host.isEmpty()) {
        settings.value(WEBSOCKET_SERVER_HOST, "").toString();
    }
    QString port = settings.value(WEBSOCKET_SERVER_PORT).toString();
    QUrl url(QString("http://%1:%2/clients/%3").arg(host).arg(port).arg(m_sNewClientFileName));
}
