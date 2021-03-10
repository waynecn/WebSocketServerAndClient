#include "registerdlg.h"
#include "ui_registerdlg.h"

#include <QTextCodec>
#include <QNetworkReply>
#include <QJsonParseError>
#include <QJsonDocument>
#include <QMessageBox>
#include <QJsonObject>
#include <QCloseEvent>
#include <QSettings>

RegisterDlg::RegisterDlg(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::RegisterDlg)
{
    Qt::WindowFlags flags= this->windowFlags();
    setWindowFlags(flags&~Qt::WindowContextHelpButtonHint);
    ui->setupUi(this);

    m_pAccessManager = new QNetworkAccessManager();
    connect(m_pAccessManager, SIGNAL(finished(QNetworkReply *)), this, SLOT(replyFinished(QNetworkReply *)));
}

RegisterDlg::~RegisterDlg()
{
    delete ui;
}

void RegisterDlg::on_okBtn_clicked()
{
    QString pwd1 = ui->passwordLineEdit->text();
    QString pwd2 = ui->confirmPasswordLineEdit->text();
    QString auth = ui->authLineEdit->text();
    if (pwd1 != pwd2) {
        QString msg = "两次输入的密码不一样";
        QMessageBox box;
        box.setWindowTitle("警告");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        return;
    }
    if (auth.isEmpty()) {
        QString msg = "授权码不能为空";
        QMessageBox box;
        box.setWindowTitle("警告");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        return;
    }
    if (auth != "10086") {
        QString msg = "授权码有误，请联系开发者QQ:635864540";
        QMessageBox box;
        box.setWindowTitle("警告");
        box.setText(msg);
        box.addButton("确定", QMessageBox::AcceptRole);
        box.exec();
        return;
    }
    m_eRequestAction = REQUEST_REGISTER;

    QSettings settings;
    QString host = settings.value(CURRENT_SERVER_HOST).toString();
    QString port = settings.value(WEBSOCKET_SERVER_PORT).toString();
    QUrl url(QString("http://%1:%2/register").arg(host).arg(port));
    QNetworkRequest req(url);
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    req.setRawHeader("token", "20200101");
    QTextCodec *codec = QTextCodec::codecForName("utf-8");
    QString loginInfo = "{\"username\":\"" + ui->userNameLineEdit->text() + "\", \"password\":\"" + pwd1 + "\", \"mobile\":\"" + ui->mobileLineEdit->text() + "\"}";
    QByteArray bData = codec->fromUnicode(loginInfo);

    m_pAccessManager->post(req, bData);
}

void RegisterDlg::on_cancelBtn_clicked()
{
    reject();
}

void RegisterDlg::replyFinished(QNetworkReply *reply) {
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

        Q_ASSERT(jsonDoc.isObject());
        bool bRet = jsonDoc["Success"].toBool();
        if (m_eRequestAction == REQUEST_REGISTER) {
            if (!bRet) {
                QString msg = "注册失败：" + jsonDoc["Msg"].toString();
                qDebug() << msg;
                QMessageBox box;
                box.setWindowTitle("提示");
                box.setText(msg);
                box.addButton("确定", QMessageBox::AcceptRole);
                box.exec();
                return;
            }

            accept();
        }
    }
}
