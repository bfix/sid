#!/bin/bash

cd runtime
clear

EXEC=""
[ -x ../bin/src ] && EXEC=../bin/src
[ -x ../src/sid ] && EXEC=../src/sid

echo "Running '${EXEC}'..."

${EXEC}

exit 0
