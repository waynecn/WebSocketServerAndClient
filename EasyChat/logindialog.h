#ifndef LOGINDIALOG_H
#define LOGINDIALOG_H

#include "common.h"

#include <QDialog>
#include <QNetworkAccessManager>

namespace Ui {
class LoginDialog;
}

class LoginDialog : public QDialog
{
    Q_OBJECT

public:
    explicit LoginDialog(QWidget *parent = nullptr);
    ~LoginDialog();

private:
    void closeEvent(QCloseEvent *e);
    void keyPressEvent(QKeyEvent *e);
    void keyReleaseEvent(QKeyEvent *e);

private slots:
    void on_loginBtn_clicked();

    void on_registerBtn_clicked();

    void replyFinished(QNetworkReply *reply);
    void downLoadNewClient();

private:
    Ui::LoginDialog         *ui;
    QNetworkAccessManager   *m_pAccessManager;

    HttpRequest             m_eRequestAction;
    bool                    m_bCtrlPressed;
    QString                 m_sNewClientFileName;

signals:
};

#endif // LOGINDIALOG_H
