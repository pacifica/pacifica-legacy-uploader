#ifndef SIMPLEBROWSER_H
#define SIMPLEBROWSER_H

#include <QtGui>
#include <QtNetwork>
#include <QtWebKit>
#include <QPointer>
#include "reusablewindow.h"

class PacificaUploaderUI: public QObject
{
	Q_OBJECT
public:
	PacificaUploaderUI();
	Q_INVOKABLE void loadLocation(QString url);
	Q_INVOKABLE QString dirUserGet();
};

class Ui_simpleBrowser;

class SimpleBrowser: public ReusableWindow
{
	Q_OBJECT
public:
	SimpleBrowser(const QUrl &url, QNetworkAccessManager *am, const char *title);
	void change_location(QUrl url);
private:
	int progress;
	Ui_simpleBrowser *uisb;
	QPointer<QWebPage> webPage;
protected slots:
	void set_progress(int p);
	void adjust_title();
	void finished_loading(bool);
	void javaScriptWindowObjectCleared();
};

#endif // SIMPLEBROWSER_H
