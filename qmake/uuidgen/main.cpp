#include<iostream>
#include<Qt>
#include<QUuid>

int main(int argc, char *argv[])
{
	std::cout << QUuid::createUuid().toString().mid(1,36).toStdString() << std::endl;
	return 0;
}
