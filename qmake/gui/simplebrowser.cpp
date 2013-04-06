#include <QtGui>
#include <QtNetwork>
#include <QtWebKit>
#include <QPointer>
#include <iostream>
#include "simplebrowser.h"
#include "ui_simplebrowser.h"

PacificaUploaderUI::PacificaUploaderUI()
{

}

void PacificaUploaderUI::loadLocation(QString url)
{
	std::cout << "Got " << url.toStdString() << "from javascript!" << std::endl;
	QDesktopServices::openUrl(QUrl(url));
}

QString PacificaUploaderUI::dirUserGet()
{
	return QFileDialog::getExistingDirectory(NULL, tr("Select a Directory"), "", QFileDialog::ShowDirsOnly);
}

SimpleBrowser::SimpleBrowser(const QUrl& url, QNetworkAccessManager *am, const char *title)
{
	uisb = new Ui::simpleBrowser();
	uisb->setupUi(this);
	this->show();
	setWindowTitle(title);
	progress = 0;
	webPage = new QWebPage();
	uisb->webView->setPage(webPage);
	webPage->setNetworkAccessManager(am);
	uisb->webView->load(url);
	connect(uisb->webView, SIGNAL(loadProgress(int)), this, SLOT(set_progress(int)));
	connect(uisb->webView, SIGNAL(loadFinished(bool)), this, SLOT(finished_loading(bool)));
	connect(webPage->mainFrame(), SIGNAL(javaScriptWindowObjectCleared()), this, SLOT(javaScriptWindowObjectCleared()));
	setUnifiedTitleAndToolBarOnMac(true);
	this->hide();
}

void SimpleBrowser::javaScriptWindowObjectCleared() {
	std::cout << "foo!!!!" << std::endl;
	webPage->mainFrame()->addToJavaScriptWindowObject("pacificaUploaderUI", new(PacificaUploaderUI));
}

void SimpleBrowser::change_location(QUrl url)
{
	uisb->webView->load(url);
	uisb->webView->setFocus();
}

void SimpleBrowser::adjust_title()
{
	if(progress <= 0 || progress >= 100)
	{
		setWindowTitle(uisb->webView->title());
	}
	else
	{
		setWindowTitle(QString("%1 (%2%)").arg(uisb->webView->title()).arg(progress));
	}
}

void SimpleBrowser::set_progress(int p)
{
	progress = p;
	adjust_title();
}

void SimpleBrowser::finished_loading(bool ok)
{
}
