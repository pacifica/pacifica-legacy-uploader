#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <malloc.h>
#include <pwd.h>
#include <sys/types.h>

int pacifica_switch_process_user(const char *userid)
{
	char *tmpptr;
	struct passwd *pwent;
	int res;
	unsigned long uid;
	uid_t realuid;
	uid = strtoul(userid, &tmpptr, 10);
	if(tmpptr == userid)
	{
		return -1;
	}
	realuid = (uid_t)uid;
	if(realuid != uid)
	{
		return -2;
	}
	pwent = getpwuid(realuid);
	if(!pwent)
	{
		return -3;
	}
	res = setregid(pwent->pw_gid, pwent->pw_gid);
	if(res)
	{
		return -4;
	}
	res = setreuid(pwent->pw_uid, pwent->pw_uid);
	if(res)
	{
		return -5;
	}
	return 0;
}

void usage()
{
	fprintf(stderr, "You must specify -u userid then program and arguments.\n");
	exit(-1);
}

int main(int argc, char *argv[])
{
	int res;
	if(argc <= 3)
	{
		usage();
	}
	if(strcmp(argv[1], "-u"))
	{
		usage();
	}
	res = pacifica_switch_process_user(argv[2]);
	if(res < 0)
	{
		return res;
	}
	res = execvp(argv[3], argv + 3);
	return res;
}
