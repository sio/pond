include ../Makefile.golang

GOTEST_TIMEOUT=5m

.PHONY: tcpflow
tcpflow:
	$@ -i lo -cDg -X /dev/null host 127.0.0.189
