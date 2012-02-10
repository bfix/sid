#!/bin/tcsh

set base = /home/WDBZ/sid
set prog = $base/sid
set logf = sid.log

while ({$prog})
	set tstamp = `date +"%Y%m%d%H%M%S"`
	mv $base/$logf $base/logs/$logf-$tstamp
end

exit 0
