#!/bin/sh
echo "db version v3.2"
if [ $# -gt 0 ]; then
    if [ $1 != "--version" ]; then
        sleep infinity;
    fi
fi
