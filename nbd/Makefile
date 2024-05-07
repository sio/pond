include ../Makefile.golang

GOTEST_TIMEOUT=5m

.PHONY: tcpflow
tcpflow:
	$@ -i lo -cDg -X /dev/null host 127.0.0.189

.PHONY: s3
s3: ## Development S3 endpoint. No security!
	docker run --rm \
		-p "127.0.0.55:55555:9000" \
		-e MINIO_ROOT_USER=access \
		-e MINIO_ROOT_PASSWORD=secret123 \
		-v "$$PWD/verity/testdata:/data/testdata:ro" \
		-it "quay.io/minio/minio:RELEASE.2022-05-26T05-48-41Z" \
		gateway nas /data

run-nbd: cache
cache:
	mkdir -p "$@"
