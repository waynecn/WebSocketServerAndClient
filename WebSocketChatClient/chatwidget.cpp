#include "chatwidget.h"
#include "ui_chatwidget.h"
#include "common.h"
#include "settingdlg.h"

#include <QJsonObject>
#include <QJsonDocument>
#include <QJsonArray>
#include <QGridLayout>
#include <QHeaderView>
#include <QSizePolicy>
#include <QFileDialog>
#include <QSettings>

ChatWidget::ChatWidget(QWidget *parent) :
    QWidget(parent),
    ui(new Ui::ChatWidget),
    m_pTextBrowser(nullptr)
{
    m_bIsMainWindow = true;
    ui->setupUi(this);

    ui->uploadFilePushButton->setEnabled(true);

    m_pTextBrowser = new MyTextBrowser();

    m_pTextBrowser->setReadOnly(true);
    m_pTextBrowser->setLineWrapMode(QTextEdit::WidgetWidth);
    QSizePolicy policy(QSizePolicy::Expanding, QSizePolicy::Expanding);
    policy.setHorizontalStretch(4);
    policy.setVerticalStretch(3);
    m_pTextBrowser->setSizePolicy(policy);
    ui->inputTextEdit->setLineWrapMode(QTextEdit::WidgetWidth);

    m_strContentTemplateWithoutLink = "<p><a style=\"color:blue\">%1:</a><br />&nbsp;&nbsp;&nbsp;&nbsp;<a>%2</a>&nbsp;&nbsp;&nbsp;&nbsp;<a style=\"color:gray\">(%3)</a></p>";
    m_strContentTemplateWithLink = "<p><a style=\"color:blue\">%1:</a><br />&nbsp;&nbsp;&nbsp;&nbsp;<a>%2</a>&nbsp;&nbsp;&nbsp;&nbsp;<a>上传文件:</a><a href=\"%3\">%4</a>&nbsp;&nbsp;&nbsp;&nbsp;<a style=\"color:gray\">(%5)</a></p>";

    int nCount = ui->showMsgTabWidget->count();
    for (int i = 0; i < nCount; ++i) {
        ui->showMsgTabWidget->removeTab(0);
    }
    ui->showMsgTabWidget->addTab(m_pTextBrowser, "主窗口");
    ui->showMsgTabWidget->setTabsClosable(true);
    ui->showMsgTabWidget->tabBar()->setTabButton(0, QTabBar::RightSide, nullptr);
    ui->showMsgTabWidget->setSizePolicy(policy);
    ui->showMsgWidget->setSizePolicy(policy);

    QSizePolicy policy2(QSizePolicy::Expanding, QSizePolicy::Expanding);
    policy2.setHorizontalStretch(4);
    policy2.setVerticalStretch(1);
    ui->widget->setSizePolicy(policy2);

    ui->onlineUsersTableWidget->setColumnCount(1);
    QTableWidgetItem *item = new QTableWidgetItem("在线用户");
    ui->onlineUsersTableWidget->setHorizontalHeaderItem(0, item);
    ui->onlineUsersTableWidget->setEditTriggers(QAbstractItemView::NoEditTriggers);
    ui->onlineUsersTableWidget->horizontalHeader()->setSectionResizeMode(QHeaderView::Stretch);

    ui->verticalSplitter->setChildrenCollapsible(false);
    ui->horizontalSplitter->setChildrenCollapsible(false);

    connect(&g_WebSocket, SIGNAL(textMessageReceived(const QString &)), this, SLOT(OnWebSocketMsgReceived(const QString &)));
    connect(ui->onlineUsersTableWidget, SIGNAL(itemDoubleClicked(QTableWidgetItem *)), this, SLOT(OnItemDoubleClicked(QTableWidgetItem *)));
    connect(ui->showMsgTabWidget, SIGNAL(tabCloseRequested(int)), this, SLOT(OnTabCloseRequested(int)));
    connect(ui->showMsgTabWidget, SIGNAL(currentChanged(int)), this, SLOT(OnCurrentChanged(int)));
    connect(ui->uploadFilePushButton, SIGNAL(clicked()), this, SLOT(OnUploadFilePushButtonClicked()));
    connect(m_pTextBrowser, SIGNAL(anchorClicked(const QUrl &)), this, SIGNAL(anchorClicked(const QUrl &)));
}

ChatWidget::~ChatWidget()
{
    delete m_pTextBrowser;
    delete ui;
}

void ChatWidget::SetSendBtnEnabled(bool b) {
    ui->sendMsgPushButton->setEnabled(b);
}

void ChatWidget::on_sendMsgPushButton_clicked()
{
    QString msg = ui->inputTextEdit->toPlainText();
    msg = msg.replace('\n', "<br />&nbsp;&nbsp;&nbsp;&nbsp;");
    int nCurIndex = ui->showMsgTabWidget->currentIndex();
    if (!msg.isEmpty() && nCurIndex == 0) {
        QJsonObject jsonObj;
        jsonObj["username"] = g_stUserInfo.strUserName;
        jsonObj["userid"] = g_stUserInfo.strUserId;
        jsonObj["message"] = msg;
        jsonObj["time"] = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
        jsonObj["filelink"] = "";
        QJsonObject jsonMsg;
        if (m_bIsMainWindow) {
            jsonMsg["message"] = jsonObj;
        } else {
            jsonMsg[g_stUserInfo.strUserId] = jsonObj;
        }
        QJsonDocument jsonDoc(jsonMsg);

        g_WebSocket.sendTextMessage(jsonDoc.toJson(QJsonDocument::Compact));
        ui->inputTextEdit->clear();
        MsgInfo msgInfo;
        msgInfo.strUserName = g_stUserInfo.strUserName;
        msgInfo.strUserId = g_stUserInfo.strUserId;
        msgInfo.strEmail = g_stUserInfo.strEmail;
        msgInfo.strMsg = msg;
        msgInfo.strTime = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
        msgInfo.fileLink = "";
        m_vecMsgInfos.push_back(msgInfo);

        m_jMessages["message"] += m_strContentTemplateWithoutLink.arg(g_stUserInfo.strUserName).arg(msg).arg(msgInfo.strTime);
        m_pTextBrowser->setHtml(m_jMessages["message"]);
        m_pTextBrowser->moveCursor(QTextCursor::End);
    } else if (!msg.isEmpty() && nCurIndex != 0) {
        QJsonObject jsonObj;
        jsonObj["username"] = g_stUserInfo.strUserName;
        jsonObj["userid"] = g_stUserInfo.strUserId;
        jsonObj["message"] = msg;
        jsonObj["time"] = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
        jsonObj["filelink"] = m_strFileLink;
        QJsonObject jsonMsg;
        jsonMsg[m_vecUserIds[nCurIndex - 1]] = jsonObj;
        QJsonDocument jsonDoc(jsonMsg);

        g_WebSocket.sendTextMessage(jsonDoc.toJson(QJsonDocument::Compact));
        ui->inputTextEdit->clear();
        MsgInfo msgInfo;
        msgInfo.strUserName = g_stUserInfo.strUserName;
        msgInfo.strUserId = g_stUserInfo.strUserId;
        msgInfo.strEmail = g_stUserInfo.strEmail;
        msgInfo.strMsg = msg;
        msgInfo.strTime = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
        msgInfo.fileLink = m_strFileLink;
        m_vecMsgInfos.push_back(msgInfo);

        MyTextBrowser *pEdit = (MyTextBrowser *)ui->showMsgTabWidget->widget(nCurIndex);
        if (m_strFileLink.isEmpty()) {
            m_jMessages[m_vecUserIds[nCurIndex - 1]] += m_strContentTemplateWithoutLink.arg(msgInfo.strUserName).arg(msg).arg(msgInfo.strTime);
        } else {
            QString fileName = msgInfo.fileLink.mid(msgInfo.fileLink.lastIndexOf('/') + 1);
            m_jMessages[m_vecUserIds[nCurIndex - 1]] += m_strContentTemplateWithLink.arg(msgInfo.strUserName).arg(msg).arg(msgInfo.fileLink).arg(fileName).arg(msgInfo.strTime);
        }
        pEdit->setHtml(m_jMessages[m_vecUserIds[nCurIndex - 1]]);
        pEdit->moveCursor(QTextCursor::End);
        connect(pEdit, SIGNAL(anchorClicked(const QUrl &)), this, SIGNAL(anchorClicked(const QUrl &)));
    }
}

void ChatWidget::keyPressEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = true;
    }

    if (m_bCtrlPressed && (e->key() == Qt::Key_Enter || e->key() == Qt::Key_Return)) {
        on_sendMsgPushButton_clicked();
    }

    if (m_bCtrlPressed && e->key() == Qt::Key_E) {
        SettingDlg dlg;
        dlg.exec();
    }
}

void ChatWidget::keyReleaseEvent(QKeyEvent *e) {
    if (e->key() == Qt::Key_Control) {
        m_bCtrlPressed = false;
    }
}

void ChatWidget::OnWebSocketMsgReceived(const QString &msg) {
    qDebug() << "receive msg:" << msg;
    if (msg.isEmpty()) {
        return;
    }
    QJsonParseError jsonErr;
    QJsonDocument jsonDoc = QJsonDocument::fromJson(msg.toUtf8(), &jsonErr);
    if (jsonErr.error != QJsonParseError::NoError) {
        qDebug() << "解析ws数据失败";
        return;
    }

    if (jsonDoc.isObject()) {
        if (jsonDoc["message"].isObject()) {
            //处理消息信息
            QJsonObject jsonMsg = jsonDoc["message"].toObject();
            QString strUserId = jsonMsg["userid"].toString();
            if (strUserId != g_stUserInfo.strUserId) {
                MsgInfo msgInfo;
                msgInfo.strUserName = jsonMsg["username"].toString();
                msgInfo.strUserId = jsonMsg["userid"].toString();
                msgInfo.strMsg = jsonMsg["message"].toString();
                msgInfo.strTime = jsonMsg["time"].toString();
                msgInfo.fileLink = jsonMsg["filelink"].toString();

                m_vecMsgInfos.push_back(msgInfo);

                if (msgInfo.fileLink.isEmpty()) {
                    m_jMessages["message"] += m_strContentTemplateWithoutLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.strTime);
                } else {
                    QString fileName = msgInfo.fileLink.mid(msgInfo.fileLink.lastIndexOf('/') + 1);
                    m_jMessages["message"] += m_strContentTemplateWithLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.fileLink).arg(fileName).arg(msgInfo.strTime);
                }
                m_pTextBrowser->setHtml(m_jMessages["message"]);
                m_pTextBrowser->moveCursor(QTextCursor::End);
                emit newMessageArrived();
                ui->showMsgTabWidget->setCurrentIndex(0);
            }
        }
        else if (jsonDoc["online"].isObject()) {
            //处理在线信息
            QJsonObject jsonOnline = jsonDoc["online"].toObject();
            QString strUserId = jsonOnline["userid"].toString();
            if (strUserId != g_stUserInfo.strUserId) {
                //将在线信息更新到右侧
                UserInfo userinfo;
                userinfo.strUserId = strUserId;
                userinfo.strUserName = jsonOnline["username"].toString();
                bool bFind = false;
                for (int i = 0; i < m_vecOnlineUsers.size(); ++i) {
                    UserInfo &user = m_vecOnlineUsers[i];
                    if (user.strUserId == strUserId) {
                        bFind = true;
                        break;
                    }
                }
                if (!bFind) {
                    m_vecOnlineUsers.push_back(userinfo);
                }
            }

            //将m_vecOnlineUsers中的内容更新到右侧
            ui->onlineUsersTableWidget->setRowCount(m_vecOnlineUsers.size());
            for (int i = 0; i < m_vecOnlineUsers.size(); ++i) {
                QTableWidgetItem *item = new QTableWidgetItem(m_vecOnlineUsers[i].strUserName);
                ui->onlineUsersTableWidget->setItem(i, 0, item);
            }
        } else if (jsonDoc[g_stUserInfo.strUserId].isObject()) {
            //私发消息 单独显示
            QJsonObject jsonMsg = jsonDoc[g_stUserInfo.strUserId].toObject();
            QString strUserId = jsonMsg["userid"].toString();
            if (strUserId != g_stUserInfo.strUserId) {
                MsgInfo msgInfo;
                msgInfo.strUserName = jsonMsg["username"].toString();
                msgInfo.strUserId = jsonMsg["userid"].toString();
                msgInfo.strMsg = jsonMsg["message"].toString();
                msgInfo.strTime = jsonMsg["time"].toString();
                msgInfo.fileLink = jsonMsg["filelink"].toString();
                //查找是否有窗口存在
                bool bExist = false;
                int nIndex = 0;
                for (int index = 0; index < ui->showMsgTabWidget->count(); ++index) {
                    QString tabName = ui->showMsgTabWidget->tabText(index);
                    if (tabName == msgInfo.strUserName) {
                        bExist = true;
                        nIndex = index;
                        break;
                    }
                }
                if (!bExist) {
                    MyTextBrowser *pEdit = new MyTextBrowser();
                    pEdit->setReadOnly(true);
                    if (msgInfo.fileLink.isEmpty()) {
                        m_jMessages[msgInfo.strUserId] += m_strContentTemplateWithoutLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.strTime);
                    } else {
                        QString fileName = msgInfo.fileLink.mid(msgInfo.fileLink.lastIndexOf('/') + 1);
                        m_jMessages[msgInfo.strUserId] += m_strContentTemplateWithLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.fileLink).arg(fileName).arg(msgInfo.strTime);
                    }
                    pEdit->setHtml(m_jMessages[msgInfo.strUserId]);
                    pEdit->moveCursor(QTextCursor::End);
                    ui->showMsgTabWidget->addTab(pEdit, msgInfo.strUserName);
                    ui->showMsgTabWidget->tabBar()->setTabButton(0, QTabBar::RightSide, nullptr);
                    ui->showMsgTabWidget->setCurrentWidget(pEdit);
                    m_vecUserIds.push_back(msgInfo.strUserId);
                    connect(pEdit, SIGNAL(anchorClicked(const QUrl &)), this, SIGNAL(anchorClicked(const QUrl &)));
                } else {
                    ui->showMsgTabWidget->setCurrentIndex(nIndex);
                    MyTextBrowser *pEdit = (MyTextBrowser *)ui->showMsgTabWidget->widget(nIndex);
                    if (msgInfo.fileLink.isEmpty()) {
                        m_jMessages[msgInfo.strUserId] += m_strContentTemplateWithoutLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.strTime);
                    } else {
                        QString fileName = msgInfo.fileLink.mid(msgInfo.fileLink.lastIndexOf('/') + 1);
                        m_jMessages[msgInfo.strUserId] += m_strContentTemplateWithLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.fileLink).arg(fileName).arg(msgInfo.strTime);
                    }
                    pEdit->setHtml(m_jMessages[msgInfo.strUserId]);
                    pEdit->moveCursor(QTextCursor::End);
                }
                emit newMessageArrived();
            }
        }
    } else if (jsonDoc.isArray()) {
        m_vecOnlineUsers.clear();
        QJsonArray array = jsonDoc.array();
        for (int i = 0; i < array.size(); ++i) {
            QJsonObject jsonObj = array[i].toObject();
            if (jsonObj["Online"].isObject()) {
                //处理在线信息
                QJsonObject jsonOnline = jsonObj["Online"].toObject();
                QString strUserId = jsonOnline["Userid"].toString();
                if (strUserId != g_stUserInfo.strUserId) {
                    //将在线信息更新到右侧
                    UserInfo userinfo;
                    userinfo.strUserId = strUserId;
                    userinfo.strUserName = jsonOnline["Username"].toString();
                    bool bFind = false;
                    for (int i = 0; i < m_vecOnlineUsers.size(); ++i) {
                        UserInfo &user = m_vecOnlineUsers[i];
                        if (user.strUserId == strUserId) {
                            bFind = true;
                            break;
                        }
                    }
                    if (!bFind) {
                        m_vecOnlineUsers.push_back(userinfo);
                    }
                }

                //将m_vecOnlineUsers中的内容更新到右侧
                ui->onlineUsersTableWidget->setRowCount(m_vecOnlineUsers.size());
                for (int i = 0; i < m_vecOnlineUsers.size(); ++i) {
                    QTableWidgetItem *item = new QTableWidgetItem(m_vecOnlineUsers[i].strUserName);
                    ui->onlineUsersTableWidget->setItem(i, 0, item);
                }
            }
        }
    }
}

void ChatWidget::OnItemDoubleClicked(QTableWidgetItem *item) {
    QString userName = item->text();
    if (userName.isEmpty()) {
        return;
    }

    int nRow = item->row();
    qDebug() << "nRow:" << nRow;
    QString userId = m_vecOnlineUsers[nRow].strUserId;
    for (int i = 0; i < m_vecUserIds.size(); ++i) {
        if (userId == m_vecUserIds[i]) {
            ui->showMsgTabWidget->setCurrentIndex(i + 1);
            return;
        }
    }
    UserInfo user;
    user.strUserId = userId;
    user.strUserName = userName;
    MyTextBrowser *pEdit = new MyTextBrowser();
    pEdit->setReadOnly(true);
    pEdit->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Expanding);
    ui->showMsgTabWidget->addTab(pEdit, userName);
    ui->showMsgTabWidget->tabBar()->setTabButton(0, QTabBar::RightSide, nullptr);
    ui->showMsgTabWidget->setCurrentWidget(pEdit);
    m_vecUserIds.push_back(userId);
    connect(pEdit, SIGNAL(anchorClicked(const QUrl &)), this, SIGNAL(anchorClicked(const QUrl &)));
}

void ChatWidget::OnTabCloseRequested(int index) {
    m_vecUserIds.removeAt(index - 1);
    ui->showMsgTabWidget->removeTab(index);
}

void ChatWidget::OnCurrentChanged(int index) {
    if (0 == index) {
        ui->uploadFilePushButton->setEnabled(true);
    } else {
        ui->uploadFilePushButton->setEnabled(true);
    }
}

void ChatWidget::OnUploadFilePushButtonClicked() {
    m_strFileLink = "";
    QFileDialog dialog;
    dialog.setFileMode(QFileDialog::ExistingFile);
    dialog.setViewMode(QFileDialog::Detail);
    QString filePath;
    if (dialog.exec()) {
        QStringList files = dialog.selectedFiles();
        if (files.size() > 0) {
            filePath = files[0];
        }
    }

    if (filePath.isEmpty()) {
        return;
    }
    QFile file(filePath);
    file.open(QIODevice::ReadOnly);
    int fileLen = file.size();
    QDataStream fileStream(&file);
    char *buff = new char[fileLen];
    memset(buff, 0, sizeof(char) * fileLen);
    fileStream.readRawData(buff, fileLen);
    file.close();

    filePath = filePath.replace("\\", "/");

    emit uploadFile(filePath);
}

void ChatWidget::OnUploadFileSuccess(QString filePath) {
    QString fileName = filePath.mid(filePath.lastIndexOf('/') + 1);
    qDebug() << "filePath:" << filePath << " fileName:" << fileName;
    QSettings setting;
    QString host = setting.value(WEBSOCKET_SERVER_HOST).toString();
    QString port = setting.value(WEBSOCKET_SERVER_PORT).toString();
    QString msg = ui->inputTextEdit->toPlainText();
    int nCurIndex = ui->showMsgTabWidget->currentIndex();
    QJsonObject jsonObj;
    jsonObj["username"] = g_stUserInfo.strUserName;
    jsonObj["userid"] = g_stUserInfo.strUserId;
    jsonObj["message"] = msg;
    jsonObj["time"] = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
    m_strFileLink = "http://" + host + ":" + port + "/uploads/" + fileName;
    jsonObj["filelink"] = m_strFileLink;
    QJsonObject jsonMsg;
    if (nCurIndex != 0) {
        jsonMsg[m_vecUserIds[nCurIndex - 1]] = jsonObj;
    } else {
        jsonMsg["message"] = jsonObj;
    }
    QJsonDocument jsonDoc(jsonMsg);

    g_WebSocket.sendTextMessage(jsonDoc.toJson(QJsonDocument::Compact));
    ui->inputTextEdit->clear();
    MsgInfo msgInfo;
    msgInfo.strUserName = g_stUserInfo.strUserName;
    msgInfo.strUserId = g_stUserInfo.strUserId;
    msgInfo.strEmail = g_stUserInfo.strEmail;
    msgInfo.strMsg = msg;
    msgInfo.strTime = QDateTime::currentDateTime().toString("yyyy-MM-dd HH:mm:ss");
    msgInfo.fileLink = m_strFileLink;
    m_vecMsgInfos.push_back(msgInfo);

    MyTextBrowser *pEdit = (MyTextBrowser *)ui->showMsgTabWidget->widget(nCurIndex);
    if (nCurIndex != 0) {
        m_jMessages[m_vecUserIds[nCurIndex - 1]] += m_strContentTemplateWithLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.fileLink).arg(fileName).arg(msgInfo.strTime);
        pEdit->setText(m_jMessages[m_vecUserIds[nCurIndex - 1]]);
    } else {
        m_jMessages["message"] += m_strContentTemplateWithLink.arg(msgInfo.strUserName).arg(msgInfo.strMsg).arg(msgInfo.fileLink).arg(fileName).arg(msgInfo.strTime);
        pEdit->setText(m_jMessages["message"]);
    }
    pEdit->moveCursor(QTextCursor::End);
    m_strFileLink.clear();
}
