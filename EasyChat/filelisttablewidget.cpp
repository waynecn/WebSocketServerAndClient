#include "filelisttablewidget.h"
#include "common.h"

#include <QDebug>
#include <QSettings>
#include <QGuiApplication>
#include <QClipboard>
#include <QMessageBox>

FileListTableWidget::FileListTableWidget(QWidget *parent) :
    QTableWidget(parent),
    m_pMenu(nullptr),
    m_pCopyLink(nullptr),
    m_pDeleteAct(nullptr)
{
    m_pMenu = new QMenu();
    m_pCopyLink = new QAction("复制链接");
    m_pDeleteAct = new QAction("删除");
    m_pMenu->addAction(m_pCopyLink);
    m_pMenu->addAction(m_pDeleteAct);

    connect(m_pCopyLink, SIGNAL(triggered(bool)), this, SLOT(OnCopyLinkTriggered(bool)));
    connect(m_pDeleteAct, SIGNAL(triggered(bool)), this, SLOT(OnDeleteActTriggered(bool)));
}

FileListTableWidget::~FileListTableWidget() {
    delete m_pMenu;
    delete m_pDeleteAct;
}

void FileListTableWidget::mousePressEvent(QMouseEvent *event) {
    if (event->button() == Qt::LeftButton) {
        QPoint pos = event->pos();
        QTableWidgetItem *item = this->itemAt(pos.x(), pos.y());
        this->clearSelection();
        this->selectRow(item->row());
    }

    //event->accept();
    QTableWidget::mousePressEvent(event);
}

void FileListTableWidget::mouseReleaseEvent(QMouseEvent *event) {
    if (event->button() == Qt::RightButton) {
        QPoint pos = event->pos();
        QTableWidgetItem *item = this->itemAt(pos.x(), pos.y());
        QTableWidgetItem *fileNameItem = this->item(item->row(), 1);
        this->clearSelection();
        this->selectRow(item->row());
        m_pMenu->exec(event->globalPos());
    }

    //event->accept();
    QTableWidget::mouseReleaseEvent(event);
}

void FileListTableWidget::OnCopyLinkTriggered(bool b) {
    QTableWidgetItem *item = currentItem();
    QTableWidgetItem *fileNameItem = this->item(item->row(), 1);

    QString fileName = fileNameItem->text();
    QSettings settings;
    QString host = settings.value(CURRENT_SERVER_HOST, "").toString();
    QString port = settings.value(WEBSOCKET_SERVER_PORT, "").toString();

    QString url = "http://" + host + ":" + port + "/uploads/" + fileName;

    QClipboard *clipboard = QGuiApplication::clipboard();
    clipboard->setText(url);
}

void FileListTableWidget::OnDeleteActTriggered(bool b) {
    QTableWidgetItem *item = currentItem();
    QTableWidgetItem *fileNameItem = this->item(item->row(), 1);

    QString fileName = fileNameItem->text();

    QMessageBox box;
    box.setWindowTitle("提示");
    box.setText("确定要删除文件:" + fileName + "吗？");
    box.addButton("确定", QMessageBox::AcceptRole);
    box.addButton("取消", QMessageBox::RejectRole);
    int nRet = box.exec();
    if (nRet != 0) {
        return;
    }

    emit deleteFile(fileName);
}
