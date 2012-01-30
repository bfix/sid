#!/bin/bash

clear

EXEC=""
[ -x ../_bin/src ] && EXEC=../_bin/src
[ -x ../src/sid ] && EXEC=../src/sid

echo "Running '${EXEC} $*' ..."

${EXEC} $*

exit 0
