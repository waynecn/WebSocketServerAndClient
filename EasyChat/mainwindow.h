#ifndef MAINWINDOW_H
#define MAINWINDOW_H

#include "common.h"
#include "chatwidget.h"
#include "progressdialog.h"
#include "settingdlg.h"

#include <QMainWindow>
#include <QSplitter>
#include <QtWebSockets/QWebSocket>
#include <QUrl>
#include <QSettings>
#include <QKeyEvent>
#include <QTableWidgetItem>
#include <QTabWidget>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QProgressDialog>
#include <QJsonArray>
#include <QTableWidgetItem>
#include <QPushButton>
#include <QMessageBox>
#include <QTime>

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
    QByteArray getFileMD5(QString filePath);
    QString fileMd5(const QString &sourceFilePath);

private slots:
    void OnWebSocketConnected();
    void OnWebSocketDisconnected();
    void OnWebSocketError(QAbstractSocket::SocketError err);
    void OnNewMessageArrived();
    void OnUploadFile(QString filePath);
    void OnAnchorClicked(const QUrl &url);
    void OnDownloadImage(QString strUrl, QString saveDir);
    void OnGetUploadFiles();
    void OnTableWidgetItemClicked(QTableWidgetItem *item);
    void OnDeleteFile(QString &fileName);
    void OnNetworkReplyFinished(QNetworkReply *reply);
    void upLoadError(QNetworkReply::NetworkError err);
    void OnUploadProgress(qint64 recved, qint64 total);
    void OnDownloadProgress(qint64 recved, qint64 total);
    void OnOpenFileDirPushed(bool b);
    void OnUploadClient(QString filepath);

private:
    Ui::MainWindow          *ui;
    SettingDlg              *m_pSettingDlg;

    HttpRequest             m_eHttpRequest;
    QString                 m_strWsUrl;
    QSettings               m_Settings;
    bool                    m_bCtrlPressed;

    QFile                   *m_pFile;
    QHttpMultiPart          *m_pMultiPart;
    ChatWidget              *m_pChatWidget;
    QNetworkAccessManager   *m_pAccessManager;
    QNetworkReply           *m_pNetworkReply;
    ProgressDialog          *m_pProgressDialog;
    QString                 m_strUploadFilePath;
    QString                 m_strDownLoadFilePath;
    QString                 m_strDownLoadImageFile;
    QPushButton             *m_pOpenFileDirPushBtn;
    QMessageBox             *m_pMsgBox;
    QTime                   m_tStart;
    qint64                  m_downLoadByte;

signals:
    void webscketDisconnected();
    void websocketConnected();
    void uploadFileSuccess(QString filePath);
    void imageDownloadFinished();
    void queryUploadFilesSuccess(QJsonArray &files);
};
#endif // MAINWINDOW_H
