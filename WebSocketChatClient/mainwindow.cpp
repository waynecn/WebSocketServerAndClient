#include "mainwindow.h"
#include "ui_mainwindow.h"
#include "settingdlg.h"

#include <QJsonObject>
#include <QJsonDocument>
#include <QJsonArray>
#include <QMessageBox>

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
    , ui(new Ui::MainWindow),
      m_pChatWidget(nullptr),
      m_pProgressDlg(nullptr)
{
    m_bCtrlPressed = false;

    ui->setupUi(this);
    m_pChatWidget = new ChatWidget();
    setCentralWidget(m_pChatWidget);

    QString host = m_Settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = m_Settings.value(WEBSOCKET_SERVER_PORT).toString();
    m_strWsUrl = QString("ws://%1:%2/ws").arg(host).arg(port);
    QUrl url(m_strWsUrl);
    g_WebSocket.open(url);

    m_pAccessManager = new QNetworkAccessManager();
    connect(m_pAccessManager, SIGNAL(finished(QNetworkReply *)), this, SLOT(replyFinished(QNetworkReply *)));

    connect(&g_WebSocket, SIGNAL(connected()), this, SLOT(OnWebSocketConnected()));
    connect(&g_WebSocket, SIGNAL(disconnected()), this, SLOT(OnWebSocketDisconnected()));
    connect(&g_WebSocket, SIGNAL(error(QAbstractSocket::SocketError)), this, SLOT(OnWebSocketError(QAbstractSocket::SocketError)));
    connect(m_pChatWidget, SIGNAL(newMessageArrived()), this, SLOT(OnNewMessageArrived()));
    connect(m_pChatWidget, SIGNAL(uploadFile(QString)), this, SLOT(OnUploadFile(QString)));
}

MainWindow::~MainWindow()
{
    if (g_WebSocket.isValid()) {
        qDebug() << "close websocket";
        g_WebSocket.close();
    }

    delete m_pChatWidget;
    if (nullptr != m_pProgressDlg) {
        delete m_pProgressDlg;
    }

    delete ui;
}

void MainWindow::keyPressEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = true;
    }
    if (m_bCtrlPressed && e->key() == Qt::Key_E) {
        SettingDlg dlg;
        dlg.exec();
    }
}

void MainWindow::keyReleaseEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = false;
    }
}

void MainWindow::OnWebSocketConnected() {
    qDebug() << "connected";
    QJsonObject jsonObj;
    jsonObj["username"] = g_stUserInfo.strUserName;
    jsonObj["userid"] = g_stUserInfo.strUserId;
    QJsonObject onlineObj;
    onlineObj["online"] = jsonObj;
    QJsonDocument jsonDoc(onlineObj);
    qint64 nRet = g_WebSocket.sendTextMessage(jsonDoc.toJson(QJsonDocument::Compact));
    qDebug() << "OnWebSocketConnect text message send:" << nRet;

    m_pChatWidget->SetSendBtnEnabled(true);
}

void MainWindow::OnWebSocketDisconnected() {
    qDebug() << "disconnected";
    m_pChatWidget->SetSendBtnEnabled(false);

    QUrl url(m_strWsUrl);
    g_WebSocket.open(url);
}

void MainWindow::OnWebSocketError(QAbstractSocket::SocketError err) {
    qDebug() << "error happened" << WEBSOCKET_ERROR_STRINGS[err + 1];
}

void MainWindow::OnNewMessageArrived() {
    QApplication::alert(this);
}

void MainWindow::OnUploadFile(QString filePath) {
    QFile file(filePath);
    file.open(QIODevice::ReadOnly);
    int fileLen = file.size();
    QDataStream fileStream(&file);
    char *buff = new char[fileLen];
    memset(buff, 0, sizeof(char) * fileLen);
    fileStream.readRawData(buff, fileLen);
    file.close();

    QString fileName = filePath.mid(filePath.lastIndexOf('/') + 1);
    QString host = m_Settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = m_Settings.value(WEBSOCKET_SERVER_PORT).toString();
    QString strUploadUrl = "http://" + host + ":" + port + "/upload?" + fileName;
    //QString strUploadUrl = "http://121.40.106.120:5139/upload";
    QUrl url(strUploadUrl);
    QNetworkRequest req(url);
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/octet-stream");
    QByteArray arr = QByteArray(buff, fileLen);
    QNetworkReply *reply = m_pAccessManager->post(req, buff);
    connect(reply, SIGNAL(error(QNetworkReply::NetworkError)), this, SLOT(upLoadError(QNetworkReply::NetworkError)));
    connect(reply, SIGNAL(uploadProgress(qint64, qint64)), this, SLOT(OnUploadProgress(qint64, qint64)));
}

void MainWindow::replyFinished(QNetworkReply *reply) {
    int statusCode = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    if(reply->error() == QNetworkReply::NoError && statusCode == 200) {
        qDebug() << "上传成功";
        m_pProgressDlg->hide();
    } else {
        qDebug() << "网络异常,请检查网络连接或服务是否正常.";
        qDebug() << "error:" << reply->error();
        QMessageBox box;
        box.setWindowTitle("提示");
        box.addButton("确认", QMessageBox::AcceptRole);
        box.setText("网络异常,请检查网络连接或服务是否正常.");
        box.exec();
    }

    reply->deleteLater();
}

void MainWindow::upLoadError(QNetworkReply::NetworkError err) {
    qDebug() << "upLoadError error:" << err;
}

void MainWindow::OnUploadProgress(qint64 recved, qint64 total) {
    if (nullptr == m_pProgressDlg) {
        m_pProgressDlg = new QProgressDialog("上传文件", "取消", 0, total);
        m_pProgressDlg->show();
    }
    qDebug() << "recved:" << recved << " total:" << total;
    m_pProgressDlg->setMaximum(total);
    m_pProgressDlg->setValue(recved);
}
