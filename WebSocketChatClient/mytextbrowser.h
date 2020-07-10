#ifndef MYTEXTBROWSER_H
#define MYTEXTBROWSER_H

#include <QTextBrowser>

class MyTextBrowser : public QTextBrowser
{
    Q_OBJECT

public:
    MyTextBrowser(QWidget *parent = nullptr);
};

#endif // MYTEXTBROWSER_H
