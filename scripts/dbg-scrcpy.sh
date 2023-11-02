#!/bin/bash





cmd=$1


if [ $cmd == 'install' ] ; then
    curl -X PUT  http://localhost:7912/scrcpy
elif [ $cmd == 'push' ]  ; then
    adb push $2 /data/local/tmp/scrcpy-server.jar
elif [ $cmd == 'start' ] ; then
    curl -X POST http://localhost:7912/services/scrcpy
elif [ $cmd == 'stop' ] ; then
    curl -X DELETE http://localhost:7912/services/scrcpy
elif [ $cmd == 'status' ] ; then
    curl -X GET http://localhost:7912/services/scrcpy
fi