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

void ProgressDialog::on_okPushButton_clicked()
{
    hide();
}
