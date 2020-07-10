#include "mytextbrowser.h"

MyTextBrowser::MyTextBrowser(QWidget *parent) :
    QTextBrowser(parent)
{
    setOpenExternalLinks(false);
    setOpenLinks(false);
}
