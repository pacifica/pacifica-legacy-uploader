using System.ServiceProcess;

namespace PacificaUploaderService
{
    static class Program
    {
        /// <summary>
        /// The main entry point for the application.
        /// </summary>
        static void Main()
        {
            ServiceBase[] ServicesToRun;
            ServicesToRun = new ServiceBase[] 
            { 
                new UploaderService() 
            };
            ServiceBase.Run(ServicesToRun);
        }
    }
}
