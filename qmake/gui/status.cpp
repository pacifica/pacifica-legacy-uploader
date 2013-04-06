#include <QDomDocument>
#include <iostream>
#include <fstream>
#include <QProcess>
#include <pacificaauth.h>
#include "status.h"
#include "simplebrowser.h"
#include "ui_pacificauploaderabout.h"
#include "reusablewindow.h"

#ifdef WIN32
#include <windows.h>
#include <winbase.h>
#else
#include <unistd.h>
#include <sys/types.h>
#endif

class UploaderAccessManager: public QNetworkAccessManager
{
	public:
		UploaderAccessManager(StatusUpdate *su);
	protected:
		StatusUpdate *_su;
		void closeEvent(QCloseEvent *event);
		QNetworkReply *createRequest(Operation op, const QNetworkRequest &req, QIODevice *data = 0);
};

UploaderAccessManager::UploaderAccessManager(StatusUpdate *su)
{
	_su = su;
}

QNetworkReply *UploaderAccessManager::createRequest(Operation op, const QNetworkRequest &req, QIODevice *data)
{
	QNetworkRequest request = req;
	QString auth = "Basic " + (_su->uploader_login_user + ":" + _su->uploader_login_passwd).toLocal8Bit().toBase64();
	request.setRawHeader("Authorization", auth.toLocal8Bit());
	QNetworkReply *reply = QNetworkAccessManager::createRequest(op, request, data);
	return reply;
}

void StatusUpdate::authentication_finished(QNetworkReply *reply)
{
	char buffer[1024];
	QString newpasswd("");
	QString tmp("");
	if(reply)
	{
		if(reply->error() == QNetworkReply::NoError)
		{
			std::ifstream file;
			QString filename = reply->readAll().trimmed();
			file.open(filename.toStdString().c_str());
			if(file.is_open())
			{
				file.getline(buffer, 1024);
				if(!file.fail())
				{
					newpasswd = QString(buffer);
					uploader_login_passwd = newpasswd;
				}
			}
		}
		reply->deleteLater();
	}
}

void StatusUpdate::uploader_login()
{
	QString user;
//FIXME unhardcode port.
	QString url("http://localhost:39999/auth/");
#ifdef WIN32
#define BUFFSIZE 512
	WCHAR buffer[BUFFSIZE];
	DWORD size = BUFFSIZE;
	char *machine;
    char *domain;

	if(!GetUserName(buffer, &size))
	{
		std::cout << "ERROR getting user" << std::endl;
	}
	user = QString::fromWCharArray(buffer, size - 1);
	if(user.indexOf('\\') == -1)
	{
        // Add case for pnl domain accounts.
        domain = getenv("USERDOMAIN");
        if (strcmp(domain, "PNL") == 0)
        {
            user = QString(domain) + QString("\\") + user;
        }
        else
        {
            machine = getenv("COMPUTERNAME");
            if(machine)
            {
                user = QString(machine) + QString("\\") + user;
            }
        }
	}
	std::cout<<"user: "<<user.toStdString()<<std::endl;
	user = user.toLower();
#else
	uid_t uid = getuid();
	user.append(QString("%1").arg(uid));
#endif
	uploader_login_user = user;
	url.append(QString("%1").arg(user));
//	std::cout << "Asking for " << url.toStdString() << std::endl;
	authmanager->get(QNetworkRequest(QUrl(url)));
}

void StatusUpdate::authentication_required(QNetworkReply *reply, QAuthenticator *authenticator)
{
//Look at reply->request headers to see if it has an auth header. if match, block and login.
	if(uploader_login_passwd != uploader_login_old_passwd)
	{
		authenticator->setUser(uploader_login_user);
		authenticator->setPassword(uploader_login_passwd);
		uploader_login_old_passwd = uploader_login_passwd;
		//std::cout << "Passing creds: " << uploader_login_user.toStdString() << " " << uploader_login_passwd.toStdString() << std::endl;
	}
	else
	{
		uploader_login();
	}
}

StatusUpdate::StatusUpdate(QSystemTrayIcon *sti)
{
	std::cout << "Entering StatusUpdate constructor" << std::endl;

	qsrand(QTime::currentTime().msec());
	authors << "<a href=\"mailto:Kevin.Fox@pnnl.gov\">Kevin Fox &lt;Kevin.Fox@pnnl.gov&gt;</a>";
	authors << "<a href=\"mailto:Nathan.Trimble@pnnl.gov\">Nathan Trimble &lt;Nathan.Trimble@pnnl.gov&gt;";
	authors << "<a href=\"mailto:Kenneth.Auberry@pnnl.gov\">Kenneth Auberry &lt;Kenneth.Auberry@pnnl.gov&gt;";
	authors << "<a href=\"mailto:David.Brown@pnnl.gov\">David Brown &lt;David.Brown@pnnl.gov&gt;";
	authors << "<a href=\"mailto:David.Cowley@pnnl.gov\">David Cowley &lt;David.Cowley@pnnl.gov&gt;";
	authors << "<a href=\"mailto:Kevin.Glass@pnnl.gov\">Kevin Glass &lt;Kevin.Glass@pnnl.gov&gt;";
    authors << "<a href=\"mailto:Craig.Allwardt@pnnl.gov\">Craig Allwardt &lt;Craig.Allwardt@pnnl.gov&gt;";

	manager = new UploaderAccessManager(this);
	uimanager = new UploaderAccessManager(this);
	authmanager = new QNetworkAccessManager(this);
	updatemanager = new QNetworkAccessManager(this);

	connect(authmanager, SIGNAL(finished(QNetworkReply*)), this, SLOT(authentication_finished(QNetworkReply*)));
	connect(manager, SIGNAL(finished(QNetworkReply*)), this, SLOT(update_finished(QNetworkReply*)));
	connect(manager, SIGNAL(authenticationRequired(QNetworkReply *, QAuthenticator *)), SLOT(authentication_required(QNetworkReply *, QAuthenticator *)));
	connect(uimanager, SIGNAL(authenticationRequired(QNetworkReply *, QAuthenticator *)), SLOT(authentication_required(QNetworkReply *, QAuthenticator *)));
	connect(updatemanager, SIGNAL(finished(QNetworkReply*)), this, SLOT(getupdate_finished(QNetworkReply*)));

	settingsbrowser = new SimpleBrowser(QUrl("about:blank"), uimanager, "Pacifica Uploader Settings");
	statusbrowser = new SimpleBrowser(QUrl("about:blank"), uimanager, "Pacifica Uploader Status");

	dialog_about = new ReusableWindow();
	ui_dialog_about = new Ui_Dialog();
	ui_dialog_about->setupUi(dialog_about);
	dialog_about->setAttribute (Qt::WA_DeleteOnClose, FALSE);

	this->sti = sti;

	connect(sti, SIGNAL(activated(QSystemTrayIcon::ActivationReason)), this, SLOT(icon_activated(QSystemTrayIcon::ActivationReason)));
	connect(sti, SIGNAL(messageClicked()), this, SLOT(messageClicked()));
	connect(this, SIGNAL(passcreds(QByteArray)), this, SLOT(get_passcreds(QByteArray)));

	ico_disconnect = new QIcon(":images/disconnect.svg");
	ico_error = new QIcon(":images/error.svg");
	ico_auth = new QIcon(":images/auth.svg");
	ico_processing = new QIcon(":images/processing.svg");
	ico_idle = new QIcon(":images/idle.svg");
	ico_uploading = new QIcon(":images/uploading.svg");;

	timer = new QTimer(this);
	connect(timer, SIGNAL(timeout()), this, SLOT(timeout()));
	timer->start(1 * 1000);

	action_about = new QAction(tr("&About"), NULL);
	connect(action_about, SIGNAL(triggered(bool)), this, SLOT(about_triggered(bool)));

	action_settings = new QAction(tr("S&ettings..."), NULL);
	connect(action_settings, SIGNAL(triggered(bool)), this, SLOT(settings_triggered(bool)));

	action_status = new QAction(tr("&Status..."), NULL);
	connect(action_status, SIGNAL(triggered(bool)), this, SLOT(status_triggered(bool)));

	action_login = new QAction(tr("&Login..."), NULL);
	connect(action_login, SIGNAL(triggered(bool)), this, SLOT(login_triggered(bool)));

	action_update = new QAction(tr("&Update Available..."), NULL);
	action_update->setVisible(false);
	connect(action_update, SIGNAL(triggered(bool)), this, SLOT(update_triggered(bool)));

	menu = new QMenu(NULL);
	menu->addAction(action_status);
	menu->addSeparator();
	menu->addAction(action_login);
	menu->addAction(action_settings);
	menu->addSeparator();
	menu->addAction(action_about);
	menu->addAction(action_update);

	sti->setContextMenu(menu);

	login_canceled = FALSE;

	QFuture<void> future = QtConcurrent::run(this, &StatusUpdate::login);
	uploader_login();

	std::cout << "Leaving StatusUpdate constructor" << std::endl;
}

void StatusUpdate::about_triggered(bool checked)
{
	QString str;
	QString newauthors = "";
	QStringList tmpauthors(authors);
	while(tmpauthors.size() > 0)
	{
		str = tmpauthors[qrand() % tmpauthors.size()];
		tmpauthors.removeOne(str);
		newauthors += str + "<br>";
	}
	ui_dialog_about->label_authors->setText(newauthors);
	ui_dialog_about->label_authors->setTextFormat(Qt::RichText);
	ui_dialog_about->label_authors->setOpenExternalLinks(true);
	dialog_about->setWindowState(Qt::WindowActive);
	dialog_about->show();
	dialog_about->raise();
}

void StatusUpdate::settings_triggered(bool checked)
{
	settingsbrowser->setWindowState(Qt::WindowActive);
	settingsbrowser->show();
	settingsbrowser->raise();
	settingsbrowser->change_location(QUrl("http://localhost:39999/config/"));
}

void StatusUpdate::status_triggered(bool checked)
{
	statusbrowser->setWindowState(Qt::WindowActive);
	statusbrowser->show();
	statusbrowser->raise();
	statusbrowser->change_location(QUrl("http://localhost:39999/status/"));
}

void StatusUpdate::login_triggered(bool checked)
{
	login_canceled = FALSE;
	wait_condition_login.wakeAll();
}

void StatusUpdate::update_triggered(bool checked)
{
	UpdateProgram();
}

void StatusUpdate::messageClicked()
{
	UpdateProgram();
}

void StatusUpdate::UpdateProgram()
{
	QString path = updatePath.replace(QString("\\"), QString("\\\\"));

	std::cout << "Attempting to update, starting " << path.toStdString() << std::endl;

	if (updateAvailable && path != "")
	{
		QProcess p;
		p.startDetached(path);
		bool started = p.waitForStarted();
		std::cout << "started = " << started << std::endl;
	}
}

void StatusUpdate::get_update()
{
	std::cout << "Entering StatusUpdate::get_update" << std::endl;
	//Initiate an HTTP GET to /update to see if the program is updateable.
	QString url("http://localhost:39999/update/");
	updatemanager->get(QNetworkRequest(QUrl(url)));
	std::cout << "Leaving StatusUpdate::get_update" << std::endl;
}

void StatusUpdate::getupdate_finished(QNetworkReply *reply)
{
	std::cout << "Entering StatusUpdate::getupdate_finished" << std::endl;
	if(reply)
	{
		if(reply->error() == QNetworkReply::NoError)
		{
			updateAvailable = false;
			updatePath = "";

			QDomDocument dom("update");
			dom.setContent(reply->readAll());

			QDomNode n = dom.firstChild();
			QDomNodeList children = n.childNodes();
			uint len = children.length();
			for (uint i = 0; i < len; i++) {
				QDomNode n = children.item(i);
				QDomElement e = n.toElement();
				if (!e.isNull()) {
					std::cout << "checking update node " << e.tagName().toStdString() << std::endl;
					if (e.tagName() == "Update") {
						if (e.text() == "true") {
							updateAvailable = true;
						}
						else {
							updateAvailable = false;
						}
					}
					else if (e.tagName() == "UpdatePath") {
						updatePath = e.text();
					}
				}
			}

			std::cout << "updateAvailable == " << updateAvailable << std::endl;
			std::cout << "updatePath == " << updatePath.toStdString() << std::endl;

			//Display an update notification if an update is available and
			//if the user has not been notified in the last 24 hours.
			//Note, this will still notifiy the user of an update on each login,
			//or whenever the program is killed and restarted.  This may or may not
			//be a problem.  Two options are two consider persisting this in the
			//status application, or sending the time the update was last checked
			//from the uploader daemon.
			if (updateAvailable &&
				updatePath != "" &&
				(lastUpdateMessageTime.secsTo(QDateTime::currentDateTime()) > 86400))
			{
				lastUpdateMessageTime = QDateTime::currentDateTime();
				sti->setToolTip(QString("Update Available"));
				sti->showMessage(QString("Pacifica Update"),
					QString("An update has been downloaded for Pacifica Uploader."),
					QSystemTrayIcon::Information,
					10000);
				action_update->setVisible(true);
				std::cout << "Update Available" << std::endl;
			}
			else if(!updateAvailable)
			{
				action_update->setVisible(false);
				std::cout << "No update available" << std::endl;
			}
		}
		else
		{
			sti->setToolTip(QString("Update check error"));
			std::cout << "Update check error" << std::endl;
		}
		reply->deleteLater();
	}
	std::cout << "Leaving StatusUpdate::getupdate_finished" << std::endl;
}

static void callback(const char *data, int size, void *user_data)
{
	QByteArray *cookiedata = (QByteArray*)user_data;
	*cookiedata += QByteArray(data, size);
}

void StatusUpdate::get_passcreds(QByteArray ba)
{
	//std::cout << "Calling out to passcreds. " << uploader_login_user.toStdString() << " " << uploader_login_passwd.toStdString() << std::endl;
	QString url("http://localhost:39999/passcreds/");
	manager->put(QNetworkRequest(QUrl(url)), ba);
}

void StatusUpdate::login()
{
	QByteArray cookiedata("");
	int res;
	mutex_login.lock();
	while(1)
	{
		wait_condition_login.wait(&mutex_login);
		if(login_canceled)
		{
			continue;
		}
		cookiedata = "";
		std::cout << "I LIVE" << std::endl;
		res = pacifica_auth(callback, &cookiedata);
		if(res)
		{
			login_canceled = TRUE;
		}
		else
		{
			emit passcreds(cookiedata);
		}
	}
	mutex_login.unlock();
}

void StatusUpdate::update_finished(QNetworkReply *reply)
{
	if(reply)
	{
		if(reply->error() == QNetworkReply::NoError)
		{
			QString str;
			QDomDocument dom("pacifica_uploader");
			dom.setContent(reply->readAll());
			//std::cout << dom.documentElement().firstChild().toText().data().toStdString() << std::endl << std::endl;
			str = dom.documentElement().firstChildElement().text();
			if(str == "idle")
			{
				sti->setToolTip(QString("Idle"));
				sti->setIcon(*ico_idle);
			}
			else if(str == "error")
			{
				sti->setToolTip(QString("Error"));
				sti->setIcon(*ico_error);
			}
			else if(str == "auth")
			{
				sti->setToolTip(QString("Authenication Required"));
				sti->setIcon(*ico_auth);
				wait_condition_login.wakeAll();
			}
			else if(str == "processing")
			{
				sti->setToolTip(QString("Processing"));
				sti->setIcon(*ico_processing);
			}
			else if(str == "uploading")
			{
				sti->setToolTip(QString("Uploading"));
				sti->setIcon(*ico_uploading);
			}
			else
			{
				std::cout << str.toStdString() << std::endl;
			}

			//std::cout << QString(reply->readAll()).toStdString() << std::endl << std::endl;
		}
		else
		{
			sti->setToolTip(QString("Connection Error"));
			sti->setIcon(*ico_disconnect);
			int status = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
			if(status == 401)
			{
				uploader_login();
			}
			std::cout << "Error: " << status << std::endl;
		}
		reply->deleteLater();
	}
}

void StatusUpdate::timeout()
{
	manager->get(QNetworkRequest(QUrl("http://127.0.0.1:39999/status/xml/")));
	this->get_update();
	return;

	//TODO - DEBUGGING code NGT 11/12/2012
	/*int r = qrand() % 5 + 1;
	switch(r)
	{
		case 1:
			sti->setIcon(*ico_disconnect);
			break;
		case 2:
			sti->setIcon(*ico_auth);
			wait_condition_login.wakeAll();
			break;
		case 3:
			sti->setIcon(*ico_processing);
			break;
		case 4:
			sti->setIcon(*ico_idle);
			break;
		case 5:
			sti->setIcon(*ico_uploading);
			break;
	}*/
}

void StatusUpdate::icon_activated(QSystemTrayIcon::ActivationReason reason)
{
	switch(reason)
	{
		case QSystemTrayIcon::Trigger:
			menu->exec(QCursor::pos());
			std::cout << "Clicked" << std::endl;
			break;
		default:
			break;
	}
}
