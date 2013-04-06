#include <QtGui>

#ifdef WIN32
#include <string.h>
#include <stdio.h>
#include <windows.h>
#include <fstream>
#include <string>
#include <QDir>
#include <QSettings>  // For creating our whitelist for people who install the application

#define sleep(x) Sleep(x * 1000)
#else
#include <unistd.h>
#endif

#include <iostream>

#include "status.h"

using namespace std;

const string WHITELIST_NAME = "whitelist";

// Gets the username based upon the platform.
string getUserName()
{
#ifdef WIN32
    return getenv("USERNAME");
#else
    return getenv("USER");
#endif
}

// Returns a path where the log file and other files are to be placed
// for the white list.  For windows if this directory does not exist
// it will create the folder.  If linux then it will not create the
// folder.
//
// returns an empty string if unsuccessful at finding a suitable path.
string getUploaderAppDataPath()
{
#ifdef WIN32
    string dataSubDir = "\\Pacifica\\Uploader\\";

    string appDataDir;
    char* appDataDirCstr = getenv("LOCALAPPDATA");
    if (appDataDirCstr == NULL)
    {
        char* userProfileCstr = getenv("USERPROFILE");
        if (userProfileCstr == NULL)
        {
            cout << "No LOCALAPPDATA or USERPROFILE environment variable found. Will not redirect cout." << endl;
            return "";
        }
        appDataDir = string(userProfileCstr) + "\\Local Settings\\Application Data";
    }
    else
    {
        appDataDir = string(appDataDirCstr);
    }

    //Get full path data directory.
    string fullDataPath = appDataDir + dataSubDir;

    //Create data directory if it doesn't exist already
    QDir dir = QDir::root();
    bool success = dir.mkpath(QString(fullDataPath.c_str()));
    if (!success)
    {
        cout << "Could not create directory " << fullDataPath << ". Will not redirect cout." << endl;
        return "";
    }
#else
    string fullDataPath = "/etc/pacifica";
#endif
    return fullDataPath;
}

#ifdef WIN32
int redirectCout()
{

	string logName = "pacificauploaderstatus.log";

    // Grab the app data directory for the uploader.
    string fullLogPath = getUploaderAppDataPath();

    if (fullLogPath.length() == 0)
    {
        cout << "Error in creating data path."<<endl;
        return 1;
    }

    //


    // append the .log file.
    fullLogPath += logName;


	//Set cout stream to log file
	ofstream* out = new ofstream(fullLogPath.c_str(), ios::out | ios::app);
	cout.rdbuf(out->rdbuf());

	cout << "redirected cout"  << endl;

	return 0;
}
#endif

// Determines whether there is a whitelist or not.
int doesWhiteListExist()
{
    QString path = QString::fromStdString(getUploaderAppDataPath());

    QDir dir(path);

    if(!dir.exists())
    {
        return 0;
    }

    path.append(QDir::separator());
    path.append(WHITELIST_NAME.c_str());

    if(QFile::exists(path))
    {
        return 1;
    }

    return 0;
}

// Determines if the current logged in user is in a whitelist
// or not. A Qsettings file looks exactly like a windows ini file an
// example for username D3M614 is as follows.  In the list either
// "1" or 1 is an acceptable value.
//      [General]
//      D3M614=1
//
// returns 0 if the user doesn't exist in the white list or if the
// whitelist itself doesn't exist.
int isUserInWhiteList()
{
    if(!doesWhiteListExist())
    {
        return 0;
    }

    QString iniFile;
    string username(getUserName());

    iniFile = QString::fromStdString(getUploaderAppDataPath());
    iniFile.append(QDir::separator());
    iniFile.append("whitelist");

    QSettings settings(iniFile, QSettings::IniFormat);

    QVariant nqv = QVariant(QVariant::Int);
    QVariant value = settings.value(QString::fromStdString(username), nqv);

    // not null then we have a whitelist value.
    // This will not throw an error if value is a string or other
    // data type.
    if(!value.isNull() && value.toInt() == 1)
    {
        return 1;
    }

    // User wasn't found.
    return 0;
}



int main(int argc, char *argv[])
{
#ifdef WIN32
	redirectCout();
#endif

    // If there is a set up white list and the
    // user is not in the white list then we need
    // to simply close this program down.  Otherwise
    // show the interface to the user.
    if (doesWhiteListExist())
    {
        cout << "White list exists" << endl;
        if (!isUserInWhiteList())
        {
            cout << getUserName() << " doesn't exist in white list. Program terminating."<< endl;
            exit(0);
            return 0;
        }
        else
        {
            cout << getUserName() << " is in white list."<< endl;
        }
    }
    else
    {
        cout << "White list does not exist." << endl;
    }

	int tray_tries = 30;
	int available;
	StatusUpdate *status;
	QApplication app(argc, argv);
    QSystemTrayIcon *trayIcon;
	QIcon icon(":images/disconnect.svg");
	available = QSystemTrayIcon::isSystemTrayAvailable();
	while(!available)
	{
		sleep(1);
		tray_tries--;
		if(tray_tries <= 0)
		{
			QMessageBox::critical(NULL, QObject::tr("System Tray"), QObject::tr("The system tray is not supported on your system."));
			return -1;
		}
		available = QSystemTrayIcon::isSystemTrayAvailable();
	}
    trayIcon = new QSystemTrayIcon(NULL);
    trayIcon->setIcon(icon);
    trayIcon->show();
    status = new StatusUpdate(trayIcon);
	return app.exec();
}
