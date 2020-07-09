#ifndef CHATWIDGET_H
#define CHATWIDGET_H

#include "common.h"

#include <QWidget>
#include <QtWebSockets/QWebSocket>
#include <QSplitter>
#include <QTableWidgetItem>
#include <QKeyEvent>
#include <QTextEdit>

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

private slots:
    void on_sendMsgPushButton_clicked();
    void OnWebSocketMsgReceived(const QString &msg);
    void OnItemDoubleClicked(QTableWidgetItem *item);
    void OnTabCloseRequested(int index);
    void OnCurrentChanged(int index);
    void OnUploadFilePushButtonClicked();

private:
    Ui::ChatWidget *ui;

    QTextEdit           *m_pTextEdit;

    bool                m_bCtrlPressed;
    QVector<MsgInfo>    m_vecMsgInfos;
    QVector<UserInfo>   m_vecOnlineUsers;
    QString             m_strFileLink;
    QString             m_strContentTemplateWithLink;
    QString             m_strContentTemplateWithoutLink;

    bool                m_bIsMainWindow;
    QVector<QString>    m_vecUserIds;   //用于保存多窗口的userId

signals:
    void newMessageArrived();
};

#endif // CHATWIDGET_H
