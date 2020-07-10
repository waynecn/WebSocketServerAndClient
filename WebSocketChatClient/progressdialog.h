#ifndef PROGRESSDIALOG_H
#define PROGRESSDIALOG_H

#include <QDialog>

namespace Ui {
class ProgressDialog;
}

class ProgressDialog : public QDialog
{
    Q_OBJECT

public:
    explicit ProgressDialog(QWidget *parent = nullptr);
    ~ProgressDialog();

    void SetProgress(int val, int total);

private slots:
    void on_okPushButton_clicked();

private:
    Ui::ProgressDialog *ui;
};

#endif // PROGRESSDIALOG_H
