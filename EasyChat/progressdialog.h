#ifndef PROGRESSDIALOG_H
#define PROGRESSDIALOG_H

#include <QDialog>
#include <QTime>

namespace Ui {
class ProgressDialog;
}

class ProgressDialog : public QDialog
{
    Q_OBJECT

public:
    explicit ProgressDialog(QWidget *parent = nullptr);
    ~ProgressDialog();

    void SetProgress(qint64 val, qint64 total);
    void SetLeftTime(qint64 timeLeft);
    void SetDownLoadSpeed(qint64 speed);

private slots:
    void on_okPushButton_clicked();

private:
    Ui::ProgressDialog *ui;
    QTime           m_tStart;
};

#endif // PROGRESSDIALOG_H
