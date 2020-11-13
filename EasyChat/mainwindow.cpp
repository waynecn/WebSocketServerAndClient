#include "mainwindow.h"
#include "ui_mainwindow.h"
#include "settingdlg.h"
#include "common.h"

#include <QJsonObject>
#include <QJsonDocument>
#include <QJsonArray>
#include <QHttpMultiPart>
#include <QFileDialog>
#include <QProcess>

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
    , ui(new Ui::MainWindow),
      m_pFile(nullptr),
      m_pMultiPart(nullptr),
      m_pChatWidget(nullptr),
      m_pProgressDialog(nullptr),
      m_pMsgBox(nullptr),
      m_pOpenFileDirPushBtn(nullptr)
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
    m_pSettingDlg = SettingDlg::GetInstance();

    m_pOpenFileDirPushBtn = new QPushButton("打开目录");
    m_pMsgBox = new QMessageBox();
    m_pMsgBox->setWindowTitle("提示");
    m_pMsgBox->addButton(m_pOpenFileDirPushBtn, QMessageBox::AcceptRole);
    m_pMsgBox->addButton("确认", QMessageBox::AcceptRole);

    connect(&g_WebSocket, SIGNAL(connected()), this, SLOT(OnWebSocketConnected()));
    connect(&g_WebSocket, SIGNAL(disconnected()), this, SLOT(OnWebSocketDisconnected()));
    connect(&g_WebSocket, SIGNAL(error(QAbstractSocket::SocketError)), this, SLOT(OnWebSocketError(QAbstractSocket::SocketError)));
    connect(m_pChatWidget, SIGNAL(newMessageArrived()), this, SLOT(OnNewMessageArrived()));
    connect(m_pChatWidget, SIGNAL(uploadFile(QString)), this, SLOT(OnUploadFile(QString)));
    connect(m_pChatWidget, SIGNAL(anchorClicked(const QUrl &)), this, SLOT(OnAnchorClicked(const QUrl &)));
    connect(m_pChatWidget, SIGNAL(downloadImage(QString, QString)), this, SLOT(OnDownloadImage(QString, QString)));
    connect(m_pChatWidget, SIGNAL(queryUploadFiles()), this, SLOT(OnGetUploadFiles()));
    connect(m_pChatWidget, SIGNAL(tableWidgetItemClicked(QTableWidgetItem *)), this, SLOT(OnTableWidgetItemClicked(QTableWidgetItem *)));
    connect(this, SIGNAL(uploadFileSuccess(QString)), m_pChatWidget, SLOT(OnUploadFileSuccess(QString)));
    connect(this, SIGNAL(imageDownloadFinished()), m_pChatWidget, SLOT(OnImageDownloadFinished()));
    connect(this, SIGNAL(queryUploadFilesSuccess(QJsonArray&)), m_pChatWidget, SLOT(OnQueryUploadFilesSuccess(QJsonArray&)));
    connect(m_pAccessManager, SIGNAL(finished(QNetworkReply*)), this, SLOT(OnNetworkReplyFinished(QNetworkReply*)));
    connect(m_pOpenFileDirPushBtn, SIGNAL(clicked(bool)), this, SLOT(OnOpenFileDirPushed(bool)));
}

MainWindow::~MainWindow()
{
    if (g_WebSocket.isValid()) {
        qDebug() << "close websocket";
        g_WebSocket.close();
    }

    delete m_pChatWidget;
    delete m_pAccessManager;
    delete m_pOpenFileDirPushBtn;
    delete m_pMsgBox;

    delete ui;
}

void MainWindow::keyPressEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = true;
    }
    if (m_bCtrlPressed && e->key() == Qt::Key_E) {
        SettingDlg *dlg = SettingDlg::GetInstance();
        dlg->exec();
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
    m_strUploadFilePath = filePath;
    m_pFile = new QFile(filePath);
    m_pFile->open(QIODevice::ReadOnly);

    QString fileName = filePath.mid(filePath.lastIndexOf('/') + 1);
    QString host = m_Settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = m_Settings.value(WEBSOCKET_SERVER_PORT).toString();
    QString uploadUrl = "http://" + host + ":" + port + "/uploads2";
    qDebug() << "uploadUrl:" << uploadUrl;

    m_pMultiPart = new QHttpMultiPart(QHttpMultiPart::FormDataType);

    QHttpPart filePart;
    filePart.setRawHeader("UserName", g_stUserInfo.strUserName.toLocal8Bit());
    filePart.setHeader(QNetworkRequest::ContentTypeHeader, QVariant("image/jpeg"));
    filePart.setHeader(QNetworkRequest::ContentDispositionHeader, QVariant(QString("form-data; name=\"file\";filename=\"" + fileName + "\";")));
    filePart.setBodyDevice(m_pFile);

    m_pMultiPart->append(filePart);

    m_eHttpRequest = REQUEST_UPLOAD_FILE;
    QUrl url(uploadUrl);
    QNetworkRequest req(url);
    req.setRawHeader("UserName", g_stUserInfo.strUserName.toLocal8Bit());
    m_pNetworkReply = m_pAccessManager->post(req, m_pMultiPart);
    connect(m_pNetworkReply, SIGNAL(error(QNetworkReply::NetworkError)), this, SLOT(upLoadError(QNetworkReply::NetworkError)));
    connect(m_pNetworkReply, SIGNAL(uploadProgress(qint64, qint64 )), this, SLOT(OnUploadProgress(qint64, qint64 )));
    if (nullptr == m_pProgressDialog) {
        m_pProgressDialog = new ProgressDialog();
    }
    m_pProgressDialog->exec();
}

void MainWindow::OnAnchorClicked(const QUrl &url) {
    m_strDownLoadFilePath.clear();
    qDebug() << "anchro clicked:" << url;
    QString strUrl = url.toString();
    QString fileName = strUrl.mid(strUrl.lastIndexOf('/') + 1);
    qDebug() << "fileName:" << fileName;

    QFileDialog fDlg;
    fDlg.setAcceptMode(QFileDialog::AcceptSave);
    fDlg.setFileMode(QFileDialog::AnyFile);
    QString saveFileDir = m_Settings.value("SAVE_FILE_DIR", "C:/").toString();
    fileName = fDlg.getSaveFileName(this, "Save File", saveFileDir + fileName);
    qDebug() << "fileName:" << fileName;
    if (fileName.isEmpty()) {
        return;
    }

    m_eHttpRequest = REQUEST_DOWNLOAD_FILE;
    m_strDownLoadFilePath = fileName;
    saveFileDir = m_strDownLoadFilePath.mid(0, m_strDownLoadFilePath.lastIndexOf("/") + 1);
    qDebug() << "saveFileDir:" << saveFileDir;
    m_Settings.setValue("SAVE_FILE_DIR", saveFileDir);

    QNetworkRequest req(url);
    QNetworkReply *downloadReply = m_pAccessManager->get(req);
    connect(downloadReply, SIGNAL(downloadProgress(qint64, qint64)), this, SLOT(OnDownloadProgress(qint64, qint64)));
}

void MainWindow::OnDownloadImage(QString strUrl, QString saveDir) {
    m_strDownLoadImageFile.clear();
    QString fileName = strUrl.mid(strUrl.lastIndexOf('/') + 1);

    m_eHttpRequest = REQUEST_DOWNLOAD_IMAGE;
    m_strDownLoadImageFile = saveDir + fileName;

    QUrl url(strUrl);
    QNetworkRequest req(url);
    m_pAccessManager->get(req);
}

void MainWindow::OnGetUploadFiles() {
    m_eHttpRequest = REQUEST_GET_UPLOAD_FILES;

    QString host = m_Settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = m_Settings.value(WEBSOCKET_SERVER_PORT).toString();
    QString fileListUrl = "http://" + host + ":" + port + "/uploadfiles2";
    QUrl url(fileListUrl);
    QNetworkRequest req(url);
    req.setRawHeader("token", "20200101");
    m_pAccessManager->get(req);
}

void MainWindow::OnTableWidgetItemClicked(QTableWidgetItem *item) {
    QString fileName = item->text();
    qDebug() << "ready to download file:" << fileName;

    QString host = m_Settings.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = m_Settings.value(WEBSOCKET_SERVER_PORT).toString();
    QString fileUrl = "http://" + host + ":" + port + "/uploads/" + fileName;
    QUrl url(fileUrl);
    OnAnchorClicked(url);
}

void MainWindow::OnNetworkReplyFinished(QNetworkReply *reply) {
    int statusCode = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    if(reply->error() == QNetworkReply::NoError && statusCode == 200) {
        if (m_eHttpRequest == REQUEST_UPLOAD_FILE) {
            qDebug() << "上传成功";
            m_pProgressDialog->accept();
            emit uploadFileSuccess(m_strUploadFilePath);
            m_strUploadFilePath.clear();
            m_pFile->close();
            delete m_pFile;
            m_pFile = nullptr;
            delete m_pMultiPart;
            m_pMultiPart = nullptr;
        } else if (m_eHttpRequest == REQUEST_DOWNLOAD_FILE) {
            if (m_strDownLoadFilePath.isEmpty()) {
                return;
            }
            QFile file(m_strDownLoadFilePath);
            if (file.open(QIODevice::WriteOnly | QIODevice::Truncate)) {
                file.write(reply->readAll());
            }
            file.close();
            qDebug() << "下载完成";
            m_pProgressDialog->accept();
            m_pMsgBox->setText(QString("文件下载完成,保存至:%1").arg(m_strDownLoadFilePath));
            m_pMsgBox->exec();
            m_strDownLoadFilePath.clear();
        } else if (m_eHttpRequest == REQUEST_DOWNLOAD_IMAGE) {
            if (m_strDownLoadImageFile.isEmpty()) {
                return;
            }
            QFile file(m_strDownLoadImageFile);
            if (file.open(QIODevice::WriteOnly | QIODevice::Truncate)) {
                file.write(reply->readAll());
            }
            file.close();
            qDebug() << "图片下载完成:" << m_strDownLoadImageFile;
            m_strDownLoadImageFile.clear();
            emit imageDownloadFinished();
        } else if (m_eHttpRequest == REQUEST_GET_UPLOAD_FILES) {
            QByteArray baData = reply->readAll();
            QJsonParseError jsonErr;
            QJsonDocument jsonDoc = QJsonDocument::fromJson(baData, &jsonErr);
            if (jsonErr.error != QJsonParseError::NoError) {
                QString msg = "解析响应数据发生错误";
                qDebug() << msg;
                QMessageBox box;
                box.setWindowTitle("EasyChat");
                box.setText(msg);
                box.addButton("确定", QMessageBox::AcceptRole);
                box.exec();
                reply->deleteLater();
                return;
            }
            bool bRet = jsonDoc["Success"].toBool();
            if (!bRet) {
                QString msg = "解析响应数据发生错误";
                qDebug() << msg;
                QMessageBox box;
                box.setWindowTitle("EasyChat");
                box.setText(msg);
                box.addButton("确定", QMessageBox::AcceptRole);
                box.exec();
                reply->deleteLater();
                return;
            }
            QJsonArray files = jsonDoc["Files"].toArray();
            emit queryUploadFilesSuccess(files);
        }
    } else {
        QString msg = "网络异常,请检查网络连接或服务是否正常.";
        qDebug() << msg << " error:" << reply->error();
        QMessageBox box;
        box.setWindowTitle("提示");
        box.addButton("确认", QMessageBox::AcceptRole);
        box.setText(msg);
        box.exec();
    }
    reply->deleteLater();
}

void MainWindow::upLoadError(QNetworkReply::NetworkError err) {
    qDebug() << "upLoadError:" << err;
}

void MainWindow::OnUploadProgress(qint64 recved, qint64 total) {
    if (nullptr == m_pProgressDialog) {
        m_pProgressDialog = new ProgressDialog();
    }

    if (m_pProgressDialog->isHidden() && recved < total) {
        m_pProgressDialog->exec();
    }

    m_pProgressDialog->SetProgress(recved, total);
}

void MainWindow::OnDownloadProgress(qint64 recved, qint64 total) {
    if (nullptr == m_pProgressDialog) {
        m_pProgressDialog = new ProgressDialog();
    }

    if (m_pProgressDialog->isHidden() && recved < total) {
        m_pProgressDialog->exec();
    }

    m_pProgressDialog->SetProgress(recved, total);
}

void MainWindow::OnOpenFileDirPushed(bool b) {
    int index = m_strDownLoadFilePath.lastIndexOf('\\');
    if (index == -1) {
        index = m_strDownLoadFilePath.lastIndexOf('/');
    }
    if (index == -1) {
        return;
    }
    QString file = m_strDownLoadFilePath;
    file = file.replace('/', '\\');
    QString cmd = "explorer /e,/select," + file;
    QProcess proc;
    proc.execute(cmd);
    proc.close();
}
