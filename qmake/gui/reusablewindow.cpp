#include "reusablewindow.h"

void ReusableWindow::closeEvent(QCloseEvent *event)
{
	event->ignore();
	this->hide();
}
