#ifndef SETTINGDLG_H
#define SETTINGDLG_H

#include "common.h"

#include <QDialog>

namespace Ui {
class SettingDlg;
}

class SettingDlg : public QDialog
{
    Q_OBJECT

public:
    explicit SettingDlg(QWidget *parent = nullptr);
    ~SettingDlg();

private slots:
    void on_okBtn_clicked();

    void on_cancelBtn_clicked();

private:
    Ui::SettingDlg *ui;
};

#endif // SETTINGDLG_H
