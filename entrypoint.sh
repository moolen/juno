#!/bin/sh
set +x
ulimit -l unlimited
ulimit -a unlimited
/juno "$@"
