






#!/bin/bash





cmd=$1


if [ $cmd == 'port' ] ; then
    adb shell netstat -lp | grep scrcpy
fi