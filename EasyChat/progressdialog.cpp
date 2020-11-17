#include "progressdialog.h"
#include "ui_progressdialog.h"

ProgressDialog::ProgressDialog(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::ProgressDialog)
{
    ui->setupUi(this);

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
    ui->leftTime->setText(QString("%1秒").arg(leftSec));
}

void ProgressDialog::on_okPushButton_clicked()
{
    accept();
}
