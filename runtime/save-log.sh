#!/bin/tcsh

set base = /home/WDBZ/sid
set logf = sid.log

set tstamp = `date +"%Y%m%d%H%M%S"`
echo $tstamp
mv $base/$logf $base/logs/$logf-$tstamp

exit 0
