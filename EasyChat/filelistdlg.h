#ifndef FILELISTDLG_H
#define FILELISTDLG_H

#include "filelisttablewidget.h"

#include <QDialog>
#include <QJsonArray>
#include <QTableWidgetItem>
#include <QMenu>
#include <QAction>
#include <QMouseEvent>

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
    void deleteFile(QString &fileName);
};

#endif // FILELISTDLG_H
