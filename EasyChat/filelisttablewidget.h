#ifndef FILELISTTABLEWIDGET_H
#define FILELISTTABLEWIDGET_H

#include <QTableWidget>
#include <QMenu>
#include <QAction>
#include <QMouseEvent>

class FileListTableWidget : public QTableWidget
{
    Q_OBJECT

public:
    FileListTableWidget(QWidget *parent = nullptr);
    ~FileListTableWidget();

    void mousePressEvent(QMouseEvent *event) override;
    void mouseReleaseEvent(QMouseEvent *event) override;

public slots:
    void OnCopyLinkTriggered(bool b);
    void OnDeleteActTriggered(bool b);

private:
    QMenu       *m_pMenu;
    QAction     *m_pCopyLink;   //复制链接
    QAction     *m_pDeleteAct;  //删除选中文件

signals:
    void deleteFile(QString &fileName);
};

#endif // FILELISTTABLEWIDGET_H
