
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
		
		// String list of arguments that are going to be passed to 
		// the uploaderd process.
		QStringList args;
				
		// build up path to the uploader process
		QString basePath = app->applicationDirPath();
				
		QString uploaderdPath = basePath + "/pacificauploaderd.exe";
		logMessage(QString("Starting Service\nProcess: " + uploaderdPath));
				
		// Create the new uploader process.
		process = new QProcess();
		// Assuming this is run from windows\system we need
		// to change the working directory to where the exe is.
		process->setWorkingDirectory(basePath);
		
		// we are in developer mode unless the service is in the
		// program files.
		if (basePath.contains("program files", Qt::CaseInsensitive))
		{
			logMessage(QString("starting using -system true"));
			args << "-system" << "true";
		}
		else
		{
			logMessage(QString("starting using: /devel"));
			args << "-devel" << "-basedir" << QString(basePath) + "/devel";
		}
		
		// Start the process here.
		process->start(uploaderdPath, args);
		
		/*
		createEvents();
		
		process->setStandardErrorFile(basePath + QString("/stderr.log"));
		process->setStandardOutputFile(basePath + QString("/stdout.log"));
		
		process->start(uploaderdPath, args );
		*/
		
		if(!process->waitForStarted())
		{
			logMessage("Error starting pacificauploaderd");
			resetProcess();
		}
		else
		{
			logMessage(QString("Process id: ") + QString::number((long)process->pid()));
		}
		
		return process != NULL;
	}
	
	void createEvents()
	{	
		// Registering the meta type for the enumerations will allow us to use
		// them in the slot/signals below.
		qRegisterMetaType<QProcess::ProcessError>("QProcess::ProcessError");
		qRegisterMetaType<QProcess::ExitStatus>("QProcess::ExitStatus");
		qRegisterMetaType<QProcess::ProcessState>("QProcess::ProcessState");
		
		QObject::connect(dynamic_cast<QObject*>(process), SIGNAL(onProcessError(QProcess::ProcessError)),
						 dynamic_cast<QObject*>(this), SLOT(onProcessError(QProcess::ProcessError)));
						 
		QObject::connect(dynamic_cast<QObject*>(process), SIGNAL(onProcessFinished(int, QProcess::ExitStatus)),
						 dynamic_cast<QObject*>(this), SLOT(onProcessFinished(int, QProcess::ExitStatus)));
		
		QObject::connect(dynamic_cast<QObject*>(process), SIGNAL(stateChanged(QProcess::ProcessState)),
						 dynamic_cast<QObject*>(this), SLOT(onProcessStateChanged(QProcess::ProcessState)));
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
	
	void onProcessStateChanged ( QProcess::ProcessState newState )
	{
		QString output = QString("Process State Changed: ");
		switch(newState)
		{
			case QProcess::NotRunning:
				output += "Not Running";
				break;
			case QProcess::Starting:
				output += "Starting";
				break;
			case QProcess::Running:
				output += "Running";
				break;
		}		
		
		logMessage(output);
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
		if(!process->waitForFinished(500))
		{
			logMessage(QString("Killing process: ") + QString::number((long)process->pid()));
			process->kill();
		}
		resetProcess();
	}
	
	void stop()
	{
		logMessage(QString("Stopping service"));
		stopUploaderdProcess();
	}


private:
	// The uploader process.
	QProcess *process;
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
