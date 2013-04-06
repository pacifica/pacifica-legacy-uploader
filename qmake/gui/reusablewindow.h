#ifndef REUSABLEWINDOW_H
#define REUSABLEWINDOW_H

#include <QtGui>

class ReusableWindow: public QMainWindow
{
	protected:
		void closeEvent(QCloseEvent *event);
};

#endif // REUSABLEWINDOW_H
