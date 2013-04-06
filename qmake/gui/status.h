#ifndef STATUSUPDATE_H
#define STATUSUPDATE_H

#include <QtGui>
#include <QtNetwork>

class Ui_Dialog;
class SimpleBrowser;

class StatusUpdate: public QObject
{
	Q_OBJECT
public:
	StatusUpdate(QSystemTrayIcon *sti);
	QString uploader_login_user;
	QString uploader_login_passwd;
private:
	bool updateAvailable;
	QString updatePath;
	QDateTime lastUpdateMessageTime;
	QString uploader_login_old_passwd;
	QMainWindow *dialog_about;
	Ui_Dialog *ui_dialog_about;
	QTimer *timer;
	QSystemTrayIcon *sti;
	QIcon *ico_disconnect;
	QIcon *ico_error;
	QIcon *ico_auth;
	QIcon *ico_processing;
	QIcon *ico_idle;
	QIcon *ico_uploading;
	QMenu *menu;
	QAction *action_about;
	QAction *action_settings;
	QAction *action_status;
	QAction *action_login;
	QAction *action_update;
	QMutex mutex_login;
	QWaitCondition wait_condition_login;
	QNetworkAccessManager *manager;
	QNetworkAccessManager *uimanager;
	QNetworkAccessManager *authmanager;
	QNetworkAccessManager *updatemanager;
	QStringList authors;
	SimpleBrowser *settingsbrowser;
	SimpleBrowser *statusbrowser;
	bool login_canceled;
	void login();
	void uploader_login();
	void get_update();
	void UpdateProgram();
protected slots:
	void timeout();
	void update_finished(QNetworkReply *);
	void icon_activated(QSystemTrayIcon::ActivationReason reason);
	void settings_triggered(bool checked);
	void status_triggered(bool checked);
	void login_triggered(bool checked);
	void update_triggered(bool checked);
	void about_triggered(bool checked);
	void messageClicked();
	void authentication_required(QNetworkReply * reply, QAuthenticator *authenticator);
	void authentication_finished(QNetworkReply *reply);
	void get_passcreds(QByteArray ba);
	void getupdate_finished(QNetworkReply* reply);
signals:
	void passcreds(QByteArray ba);
};

#endif // STATUSUPDATE_H
