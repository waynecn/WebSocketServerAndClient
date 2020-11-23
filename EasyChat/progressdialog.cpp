#include "progressdialog.h"
#include "ui_progressdialog.h"

ProgressDialog::ProgressDialog(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::ProgressDialog)
{
    ui->setupUi(this);
    ui->okPushButton->hide();

    setWindowFlags(windowFlags() & Qt::WindowCloseButtonHint);
}

ProgressDialog::~ProgressDialog()
{
    delete ui;
}


void ProgressDialog::SetProgress(qint64 val, qint64 total) {
    ui->progressBar->setMaximum(total);
    ui->progressBar->setValue(val);
    if (val >= total) {
        accept();
    }
}

void ProgressDialog::SetLeftTime(qint64 timeLeft) {
    int leftSec = timeLeft / 1000;
    int leftHour = 0;
    int leftMiniute = 0;
    QString str = "";
    if (leftSec > 3600) {
        leftHour = leftSec / 3600;
        str = QString("%1小时%2分钟%3秒").arg(leftHour).arg((leftSec - leftHour * 3600) / 60).arg(leftSec % 60);
    } else if (leftSec <= 3600 && leftSec > 60) {
        leftMiniute = leftSec / 60;
        str = QString("%1分钟%2秒").arg(leftMiniute).arg(leftSec % 60);
    } else {
        str = QString("%1秒").arg(leftSec);
    }

    ui->leftTime->setText(str);
}

void ProgressDialog::on_okPushButton_clicked()
{
    accept();
}
