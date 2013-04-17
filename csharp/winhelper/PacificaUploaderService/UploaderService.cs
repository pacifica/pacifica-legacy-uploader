using System;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using System.ServiceProcess;
using System.Text;

namespace PacificaUploaderService
{
    public class UploaderService : ServiceBase
    {
        private const string uploaderFileName = "pacificauploaderd.exe";
        private const string uploaderArguments = "-system";
        private const string uploaderBaseDir = "Pacifica\\Uploader";
        private const string uploaderLogFileName = "pacificauploaderservice.log";

        private Process uploader;
        private StreamWriter uploaderOutput;
        private readonly object uploaderOutputLock = new object();
        private string uploaderLogFileFullPath;

        public UploaderService()
        {
            ServiceName = "Pacifica Uploader";
        }

        protected override void Dispose(bool disposing)
        {
            if (disposing)
            {
                if (uploaderOutput != null)
                {
                    uploaderOutput.Dispose();
                }
            }
            base.Dispose(disposing);
        }

        public new void Dispose()
        {
            Dispose(true);
        }

        protected override void OnStart(string[] args)
        {
            if (!StartUploader())
            {
                EventLog.WriteEntry(ServiceName, "Failed to start uploader daemon.",
                    EventLogEntryType.Error);
            }
        }

        protected override void OnStop()
        {
            StopUploader();
        }

        private bool StartUploader()
        {
            uploader = new Process();
            var path = GetUploaderFileName();
            if (string.IsNullOrEmpty(path))
            {
                return false;
            }

            SetupUploaderOutput();

            uploader.StartInfo.FileName = path;
            uploader.StartInfo.Arguments = uploaderArguments;
            uploader.StartInfo.UseShellExecute = false;
            uploader.StartInfo.RedirectStandardOutput = true;
            uploader.StartInfo.RedirectStandardError = true;
            uploader.OutputDataReceived += OnOutputDataReceived;
            uploader.ErrorDataReceived += OnErrorDataReceived;
            bool result = false;
            try
            {
                result = uploader.Start();
                uploader.BeginOutputReadLine();
                uploader.BeginErrorReadLine();
                if (!result)
                {
                    EventLog.WriteEntry(ServiceName,
                        string.Format("{0} failed to start.", uploader.StartInfo.FileName),
                        EventLogEntryType.Error);
                }
            }
            catch (Exception ex)
            {
                result = false;
                EventLog.WriteEntry(ServiceName,
                    string.Format("{0} failed to start. Exception caught type {1}, message {2}, stacktrace {3}",
                        uploader.StartInfo.FileName, ex.GetType().ToString(), ex.Message, ex.StackTrace),
                    EventLogEntryType.Error);

            }
            return result;
        }

        private void SetupUploaderOutput()
        {
            lock (uploaderOutputLock)
            {
                try
                {
                    var logFileDir = Path.Combine(
                        Environment.GetFolderPath(Environment.SpecialFolder.CommonApplicationData),
                        uploaderBaseDir);
                    if (!Directory.Exists(logFileDir))
                    {
                        Directory.CreateDirectory(logFileDir);
                    }
                    uploaderLogFileFullPath = Path.Combine(
                        logFileDir,
                        uploaderLogFileName);
                    if (!string.IsNullOrEmpty(uploaderLogFileFullPath))
                    {
                        EventLog.WriteEntry(ServiceName,
                            string.Format("{0} is uploader log file.", uploaderLogFileFullPath),
                            EventLogEntryType.Information);
                    }
                    if (uploaderOutput != null && uploaderOutput.BaseStream.CanWrite)
                    {
                        return;
                    }
                    if (uploaderOutput != null)
                    {
                        uploaderOutput.Dispose();
                        uploaderOutput = null;
                    }
                    uploaderOutput = new StreamWriter(uploaderLogFileFullPath, true, Encoding.UTF8);
                }
                catch (Exception ex)
                {
                    EventLog.WriteEntry(ServiceName,
                        string.Format("Failed to create {0}. Exception caught type {1}, message {2}, stacktrace {3}",
                        uploaderLogFileFullPath, ex.GetType().ToString(), ex.Message, ex.StackTrace),
                        EventLogEntryType.Error);
                }
            }
        }

        private void TearDownUploaderOutput()
        {
            lock (uploaderOutputLock)
            {
                if (uploaderOutput != null)
                {
                    uploaderOutput.Dispose();
                }
            }
        }

        private void OnOutputDataReceived(object sender, DataReceivedEventArgs e)
        {
            WriteLine(e.Data);
        }

        private void OnErrorDataReceived(object sender, DataReceivedEventArgs e)
        {
            WriteLine(e.Data);
        }

        private void WriteLine(string line)
        {
            lock (uploaderOutputLock)
            {
                try
                {
                    uploaderOutput.WriteLine(line);
                    uploaderOutput.Flush();
                }
                catch (Exception ex)
                {
                    EventLog.WriteEntry(ServiceName,
                        string.Format("Failed to write to {0}. Exception caught type {1}, message {2}, stacktrace {3}",
                        uploaderLogFileFullPath, ex.GetType().ToString(), ex.Message, ex.StackTrace),
                        EventLogEntryType.Error);
                }
            }
        }

        private void StopUploader()
        {
            try
            {
                //TODO - We should come up with a better mechanism to tell the uploader daemon to shutdown.
                //We may have to "roll our own", something custom like a pipe, socket, etc.
                //Unix signals would be nice in theory, but I am unsure if it'll work on Windows.
                //NGT 2/14/2012
                uploader.Kill();
                uploader.Dispose();
                uploader = null;
                TearDownUploaderOutput();
            }
            catch (Exception ex)
            {
                EventLog.WriteEntry(ServiceName,
                    string.Format("{0} failed to terminate.  Exception caught type {1}, message {2}, stacktrace {3}.",
                        uploader.StartInfo.FileName, ex.GetType().ToString(), ex.Message, ex.StackTrace),
                    EventLogEntryType.Error);
            }
        }

        private string GetUploaderFileName()
        {
            var fullPath = Assembly.GetExecutingAssembly().Location;
            FileInfo fi = new FileInfo(fullPath);
            fullPath = Path.Combine(fi.DirectoryName, uploaderFileName);
            if (!File.Exists(fullPath))
            {
                EventLog.WriteEntry(ServiceName,
                    string.Format("{0} does not exist.", fullPath),
                    EventLogEntryType.Error);
                return null;
            }
            return fullPath;
        }
    }
}
