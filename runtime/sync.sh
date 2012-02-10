#!/bin/bash

#######################################################################
# sync to/from survey server
#######################################################################

LOCAL=/usr/prj/Y_Security/HiddenServer/SID/runtime
REMOTE=WDBZ@survey-server:/home/WDBZ/sid

if [ "$1" == "to" ]; then
	rsync -vaHx --progress --exclude=./logs --exclude=./sync.sh ${LOCAL}/. ${REMOTE}
elif [ "$1" == "from" ]; then
	rsync -vaHx --progress ${REMOTE}/logs ${LOCAL}/logs
else
	echo "Unknown direction '$1': [to|from] expected."
fi

exit 0
