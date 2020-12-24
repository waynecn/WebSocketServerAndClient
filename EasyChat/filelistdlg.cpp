#include "filelistdlg.h"
#include "ui_filelistdlg.h"

#include <QJsonArray>
#include <QTableWidgetItem>
#include <QPushButton>
#include <QDebug>
#include <QJsonObject>

FileListDlg::FileListDlg(QWidget *parent) :
    QDialog(parent),
    ui(new Ui::FileListDlg)
{
    Qt::WindowFlags flags= this->windowFlags();
    setWindowFlags(flags&~Qt::WindowContextHelpButtonHint);
    ui->setupUi(this);

    ui->tableWidget->setColumnCount(5);
    ui->tableWidget->horizontalHeader()->setSectionResizeMode(QHeaderView::Stretch);
    ui->tableWidget->horizontalHeader()->setSectionResizeMode(0, QHeaderView::ResizeToContents);
    ui->tableWidget->horizontalHeader()->setSectionResizeMode(1, QHeaderView::ResizeToContents);
    QStringList headerLables;
    headerLables.push_back("序号");
    headerLables.push_back("文件名");
    headerLables.push_back("文件大小");
    headerLables.push_back("上传者");
    headerLables.push_back("操作");
    ui->tableWidget->setHorizontalHeaderLabels(headerLables);

    connect(ui->tableWidget, SIGNAL(itemClicked(QTableWidgetItem *)), this, SLOT(OnItemClicked(QTableWidgetItem *)));
    connect(this, SIGNAL(onDownLoadItem(QTableWidgetItem *)), this, SIGNAL(tableWidgetItemClicked(QTableWidgetItem *)));
    connect(ui->tableWidget, SIGNAL(deleteFile(QString &)), this, SIGNAL(deleteFile(QString &)));
}

FileListDlg::~FileListDlg()
{
    delete ui;
}

void FileListDlg::OnQueryUploadFilesSuccess(QJsonArray &files) {
    this->resize(this->size());
    int size = files.size();
    ui->tableWidget->setRowCount(size);

    for (int i = 0; i < size; ++i) {
        QJsonObject obj = files[i].toObject();
        //qDebug() << "Obj:" << obj;
        QTableWidgetItem *item = new QTableWidgetItem(obj["FileName"].toString());
        ui->tableWidget->setItem(i, 0, new QTableWidgetItem(QString("%1").arg(i + 1)));
        ui->tableWidget->setItem(i, 1, item);
        ui->tableWidget->setItem(i, 2, new QTableWidgetItem(QString("%1").arg(obj["FileSize"].toInt())));
        QString uploadUser = obj["UploadUser"].toObject()["Valid"].toBool() ? obj["UploadUser"].toObject()["String"].toString() : "未知";
        ui->tableWidget->setItem(i, 3, new QTableWidgetItem(uploadUser));
        ui->tableWidget->setItem(i, 4, new QTableWidgetItem("下载"));
    }
}

void FileListDlg::OnItemClicked(QTableWidgetItem *item) {
    if (item->column() == 4) {
        QTableWidgetItem * fileNameItem = ui->tableWidget->item(item->row(), 1);

        emit onDownLoadItem(fileNameItem);
    }
}
