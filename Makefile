APP_NAME := adit
DOCKER_IMAGE_SRV := $(APP_NAME)_srv
DOCKER_IMAGE_CLIENT := $(APP_NAME)_client

.PHONY: test
test:
	@{ \
		trap '$(MAKE) clean-test' EXIT; \
		if ! $(MAKE) build-srv-test; then exit 1; fi; \
		if ! $(MAKE) run-srv-test; then exit 1; fi; \
		if ! $(MAKE) run-client-test; then exit 1; fi; \
	}

.PHONY: build-srv-test
build-srv-test:
	@echo "Building the Docker image for srv..."
	docker build -f docker/srv_test.dockerfile -t $(DOCKER_IMAGE_SRV) .

.PHONY: run-srv-test
run-srv-test:
	@echo "Running the Docker container..."
	docker run --rm -d -p 8080:8080 $(DOCKER_IMAGE_SRV) 

.PHONY: run-client-test
run-client-test:
	@echo "Running client test script"
	./test/client_test.sh

.PHONY: clean-test
clean-test:
	@echo "Cleaning up test files"
	rm -f ./test/100mb.file ./test/adit-client ./test/pipefile
	@echo "removing adit_srv docker container"
	for id in $$(docker ps --filter=ancestor=adit_srv:latest --format "{{.ID}}"); do docker stop $$id; done
