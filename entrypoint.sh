#!/bin/sh
set +x

if [ -n "$SET_ULIMIT" ]; then
    ulimit -l unlimited
fi

ulimit -a unlimited
/juno "$@"
