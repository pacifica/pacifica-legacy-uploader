
#include <QCoreApplication>
#include <QFile>
#include <QTextStream>
#include <QDateTime>
#include <QStringList>
#include <QDir>
#include <QSettings>
#include <QProcess>
#include <QObject>
#include "qtservice.h"

/*
	This class creates and stops the daemon pacifica-uploader.
*/
class UploaderService : public QtService<QCoreApplication>, public QObject
{
public:
    UploaderService(int argc, char **argv)
	: QtService<QCoreApplication>(argc, argv, "Pacifica Uploader Service")
    {
        setServiceDescription("Service to create and monitor Pacifica Uploader");
        //setServiceFlags(QtServiceBase::CanBeSuspended);
		process = NULL;
    }

protected:
    void start()
    {
		if(this->startUploaderdProcess())
		{
			logMessage("Uploader process started");
		}
		else
		{
			logMessage("Couldn't start uploader process!");
			exit(5000);
		}
    }
	
	bool startUploaderdProcess()
	{
		QDir dirLocation(QDir::currentPath());
		
		// Reference the application to get the installation directory.
        QCoreApplication* app = application();
		QStringList args;
		
		
		// build up path to the uploader process
		QString basePath = app->applicationDirPath();
		args << "-devel" <<"-basedir"<<basePath;
		logMessage(QString("Changing Directories"));
		//QDir::setPath(basePath);
		//dirLocation.cd(basePath);
		QString uploaderdPath = basePath + "/pacificauploaderd.exe";
		logMessage(QString("Starting Service\nProcess: " + uploaderdPath));
		logMessage(QString("Present Working Dir: " + QDir::currentPath()));
		
		// Create the new uploader process.
		process = new QProcess();
		process->setWorkingDirectory(basePath);
		createEvents();
		
		process->setStandardErrorFile(basePath + QString("/stderr.log"));
		process->setStandardOutputFile(basePath + QString("/stdout.log"));
		
		process->start(uploaderdPath, args );
		
		if(!process->waitForStarted())
		{
			logMessage("Error starting pacificauploaderd");
			delete process;
			process = NULL;
		}
		
		return process != NULL;
	}
	
	void createEvents()
	{	
		qRegisterMetaType<QProcess::ProcessError>("QProcess::ProcessError");
		qRegisterMetaType<QProcess::ExitStatus>("QProcess::ExitStatus");
		
		QObject::connect(dynamic_cast<QObject*>(process), SIGNAL(onProcessError(QProcess::ProcessError)),
						 dynamic_cast<QObject*>(this), SLOT(onProcessError(QProcess::ProcessError)));
						 
		QObject::connect(dynamic_cast<QObject*>(process), SIGNAL(onProcessFinished(int, QProcess::ExitStatus)),
						 dynamic_cast<QObject*>(this), SLOT(onProcessFinished(int, QProcess::ExitStatus)));
	}
	
	void onProcessFinished ( int exitCode, QProcess::ExitStatus exitStatus )
	{	
		QString output = QString("Process finished: ");
		if (exitStatus == QProcess::NormalExit)
		{
			output += "Normal";
		}
		else
		{
			output += "Crashed";
		}
		logMessage(output);
		resetProcess();
	}
	
	void resetProcess()
	{
		delete process;
		process = NULL;
	}
	
	/*
	
	void	readyReadStandardError ()
	void	readyReadStandardOutput ()
	void	started ()
	void	stateChanged ( QProcess::ProcessState newState )*/
	
	void onProcessError ( QProcess::ProcessError error )
	{
		QString output = QString("Error: ");
		
		switch (error)
		{
			case QProcess::Crashed:
				output += "Crashed";
				break;
			case QProcess::Timedout:
				output += "Timedout";
				break;
			case QProcess::WriteError:
				output += "WriteError";
				break;
			case QProcess::ReadError:
				output += "ReadError";
				break;
			case QProcess::UnknownError:
				output += "UnknownError";
				break;
			case QProcess::FailedToStart:
				output += "FaildToStart";
				break;
		}
		logMessage(output);
		resetProcess();
	}
	
	
	void stopUploaderdProcess()
	{
		if(process == NULL)
		{
			return;
		}
		
		process->terminate();
		if(!process->waitForFinished())
		{
			process->kill();
		}
		resetProcess();
	}
	
	void stop()
	{
		logMessage(QString("Stopping service"));
		stopUploaderdProcess();
	}

	/*
    void pause()
    {
		logMessage(QString("Pauseing Service"));
    }

    void resume()
    {
		logMessage(QString("Resuming Service"));
    }*/

private:
	//QCoreApplication *app;
	QProcess *process;
	//QFile* pFile;
    //TestService *daemon;
};


int main(int argc, char **argv)
{
	UploaderService service(argc, argv);
	return service.exec();
// #if !defined(Q_OS_WIN)
    // // QtService stores service settings in SystemScope, which normally require root privileges.
    // // To allow testing this example as non-root, we change the directory of the SystemScope settings file.
    // QSettings::setPath(QSettings::NativeFormat, QSettings::SystemScope, QDir::tempPath());
    // qWarning("(Example uses dummy settings file: %s/QtSoftware.conf)", QDir::tempPath().toLatin1().constData());
// #endif
    // HttpService service(argc, argv);
    // return service.exec();
}
