.PHONY: build build-server build-android test test-server lint lint-server lint-android

build: build-server build-android

build-server:
	cd server && go build ./...

build-android:
	cd android && ./gradlew :app:assembleDevDebug

test: test-server

test-server:
	cd server && go test ./... -cover

lint: lint-server lint-android

lint-server:
	cd server && golangci-lint run ./...

lint-android:
	cd android && ./gradlew :app:lintDevDebug
