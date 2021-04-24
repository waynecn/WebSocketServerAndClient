#ifndef CHATWIDGET_H
#define CHATWIDGET_H

#include "common.h"
#include "mytextbrowser.h"
#include "filelistdlg.h"

#include <QWidget>
#include <QtWebSockets/QWebSocket>
#include <QSplitter>
#include <QTableWidgetItem>
#include <QKeyEvent>
#include <QJsonArray>

namespace Ui {
class ChatWidget;
}

class ChatWidget : public QWidget
{
    Q_OBJECT

public:
    explicit ChatWidget(QWidget *parent = nullptr);
    ~ChatWidget();
    void SetSendBtnEnabled(bool b);

    void keyPressEvent(QKeyEvent *e);
    void keyReleaseEvent(QKeyEvent *e);
    bool isImage(QString fileName);

public slots:
    void OnUploadFileSuccess(QString filePath);
    void OnImageDownloadFinished();
    void OnQueryUploadFilesSuccess(QJsonArray &files);

private slots:
    void on_sendMsgPushButton_clicked();
    void OnWebSocketMsgReceived(const QString &msg);
    void OnItemDoubleClicked(QTableWidgetItem *item);
    void OnTabCloseRequested(int index);
    void OnCurrentChanged(int index);
    void OnUploadFilePushButtonClicked();

    void on_fileListPushButton_clicked();
    void OnUploadClient();

private:
    Ui::ChatWidget *ui;

    MyTextBrowser       *m_pTextBrowser;
    FileListDlg         *m_pFileListDlg;

    bool                m_bCtrlPressed;
    QVector<MsgInfo>    m_vecMsgInfos;
    QVector<UserInfo>   m_vecOnlineUsers;
    QString             m_strFileLink;
    QString             m_strContentTemplateWithLink;
    QString             m_strContentTemplateWithoutLink;
    QString             m_strContentTemplateWithLinkWithImage;
    QString             m_strContentTemplateWithoutLinkWithImage;
    QMap<QString, QString>  m_jMessages;

    bool                m_bIsMainWindow;
    QVector<QString>    m_vecUserIds;   //用于保存多窗口的userId

    QString             m_strImageDir;

signals:
    void newMessageArrived();
    void uploadFile(QString filePath);
    void anchorClicked(const QUrl &url);
    void downloadImage(QString strUrl, QString saveDir);
    void queryUploadFiles();
    void queryUploadFilesSuccess(QJsonArray &files);
    void tableWidgetItemClicked(QTableWidgetItem *item);
    void deleteFile(QString &f);
    void uploadClient();
    void uploadClient(QString filePath);
};

#endif // CHATWIDGET_H
