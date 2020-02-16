#!/usr/bin/env bash

curl -w "@curl_format.txt" -o /dev/stdout -s "localhost:5055/sds/ServiceDir/penny.prm?mode=hdr"
