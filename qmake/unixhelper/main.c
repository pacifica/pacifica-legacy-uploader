#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>

void usage()
{
	fprintf(stderr, "You must specify -p pidfilename then program and arguments.\n");
	exit(-1);
}

int main(int argc, char *argv[])
{
	FILE *fd;
	int res;
	if(argc <= 3)
	{
		usage();
	}
	if(strcmp(argv[1], "-p"))
	{
		usage();
	}
	fd = fopen(argv[2], "w");
	if(!fd)
	{
		fprintf(stderr, "Failed to open pid file.\n");
		exit(-1);
	}
	res = daemon(0, 0);
	if(res == -1)
	{
		fprintf(stderr, "Failed to daemonize. %d\n", errno);
		exit(-1);
	}
	fprintf(fd, "%d\n", getpid());
	fclose(fd);
	res = execvp(argv[3], argv + 3);
	return res;
}
