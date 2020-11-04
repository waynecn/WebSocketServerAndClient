#include "filelistdlg.h"
#include "ui_filelistdlg.h"

#include <QJsonArray>
#include <QTableWidgetItem>
#include <QPushButton>


FileListDlg::FileListDlg(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::FileListDlg)
{
    Qt::WindowFlags flags= this->windowFlags();
    setWindowFlags(flags&~Qt::WindowContextHelpButtonHint);
    ui->setupUi(this);

    ui->tableWidget->setColumnCount(3);
    ui->tableWidget->horizontalHeader()->setHorizontalScrollMode(QAbstractItemView::ScrollPerItem);
    QStringList headerLables;
    headerLables.push_back("序号");
    headerLables.push_back("文件名");
    headerLables.push_back("操作");
    ui->tableWidget->setHorizontalHeaderLabels(headerLables);

    connect(ui->tableWidget, SIGNAL(itemClicked(QTableWidgetItem *)), this, SLOT(OnItemClicked(QTableWidgetItem *)));
    connect(this, SIGNAL(onDownLoadItem(QTableWidgetItem *)), this, SIGNAL(tableWidgetItemClicked(QTableWidgetItem *)));
}

FileListDlg::~FileListDlg()
{
    delete ui;
}

void FileListDlg::OnQueryUploadFilesSuccess(QJsonArray &files) {
    int size = files.size();
    ui->tableWidget->setRowCount(size);

    for (int i = 0; i < size; ++i) {
        QTableWidgetItem *item = new QTableWidgetItem(files[i].toString());
        ui->tableWidget->setItem(i, 0, new QTableWidgetItem(QString("%1").arg(i + 1)));
        ui->tableWidget->setItem(i, 1, item);
        ui->tableWidget->setItem(i, 2, new QTableWidgetItem("下载"));
    }
}

void FileListDlg::OnItemClicked(QTableWidgetItem *item) {
    if (item->column() == 2) {
        QTableWidgetItem * fileNameItem = ui->tableWidget->item(item->row(), 1);

        emit onDownLoadItem(fileNameItem);
    }
}
