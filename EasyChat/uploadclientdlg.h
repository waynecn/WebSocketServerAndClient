#ifndef UPLOADCLIENTDLG_H
#define UPLOADCLIENTDLG_H

#include <QDialog>

namespace Ui {
class UploadClientDlg;
}

class UploadClientDlg : public QDialog
{
    Q_OBJECT

public:
    explicit UploadClientDlg(QWidget *parent = nullptr);
    ~UploadClientDlg();

private:
    Ui::UploadClientDlg *ui;
};

#endif // UPLOADCLIENTDLG_H
