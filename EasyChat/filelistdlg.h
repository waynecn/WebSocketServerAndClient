#ifndef FILELISTDLG_H
#define FILELISTDLG_H

#include <QDialog>
#include <QJsonArray>
#include <QTableWidgetItem>

namespace Ui {
class FileListDlg;
}

class FileListDlg : public QDialog
{
    Q_OBJECT

public:
    explicit FileListDlg(QWidget *parent = nullptr);
    ~FileListDlg();

public slots:
    void OnQueryUploadFilesSuccess(QJsonArray &files);
    void OnItemClicked(QTableWidgetItem *item);

private:
    Ui::FileListDlg *ui;

signals:
    void tableWidgetItemClicked(QTableWidgetItem *item);
    void onDownLoadItem(QTableWidgetItem *item);
};

#endif // FILELISTDLG_H
