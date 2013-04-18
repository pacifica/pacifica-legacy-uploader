@echo off
REM This file is meant to be run from a windows command line.  It will
REM run the specific exe's listed in order and should pass or fail based
REM upone the tests.

REM cd to where the tests are

setlocal ENABLEDELAYEDEXPANSION

REM each test that should be run should be in this list.  the
REM exe must be in the build directory in order for it to work.
set teststring=archiver.test;common.test;sqlite.test

REM Do something with each substring
:stringLOOP
    REM Stop when the string is empty
    if "!teststring!" EQU "" goto END

    for /f "delims=;" %%a in ("!teststring!") do set substring=%%a

        REM Do something with the substring - 
        REM we just echo it for the purposes of demo
        echo executing !substring!
		build\!substring!.exe >!substring!.tmp
		set /p data=<!substring!.tmp
		del !substring!.tmp
		if !data! NEQ PASS (
			echo FAIL !data!
		) else (
			echo %data%
		)

REM Now strip off the leading substring
:striploop
    set stripchar=!teststring:~0,1!
    set teststring=!teststring:~1!

    if "!teststring!" EQU "" goto stringloop

    if "!stripchar!" NEQ ";" goto striploop

    goto stringloop
)

:END
endlocal