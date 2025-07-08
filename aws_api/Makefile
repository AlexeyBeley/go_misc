
SHELL := /bin/bash

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BUILD_DIR:= ${ROOT_DIR}/_build
GOENVPATH:=$(go env GOPATH)

APP_NAME := aws_api
APP_BUILD_DIR:=${BUILD_TMP_DIR}/${APP_NAME}



install-golang:
	wget https://go.dev/dl/go1.24.3.linux-amd64.tar.gz &&\
	sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz &&\
	export PATH=$PATH:/usr/local/go/bin
	//$HOME/.profile.

install-requirements intall-golang:
	sudo apt update &&\
	sudo apt -yff install dpkg-dev

test:
	echo ${BUILD_DIR}

init_build_dir:
	mkdir -p ${APP_BUILD_DIR} &&\
	mkdir -p ${APP_BUILD_DIR}/DEBIAN &&\
	mkdir -p ${APP_BUILD_DIR}/usr/local/bin &&\
	mkdir -p ${APP_BUILD_DIR}/etc/${APP_NAME} &&\
	mkdir -p ${APP_BUILD_DIR}/opt/${APP_NAME}/IamDataDir &&\
	cp ${BUILD_DIR}/control ${APP_BUILD_DIR}/DEBIAN &&\
	cp data/* ${APP_BUILD_DIR}/opt/${APP_NAME}/IamDataDir


compile: init_build_dir install-requirements compile_raw


compile_raw:
	mv ../go_prod.mod ../go.mod &&\
	go mod tidy &&\
	GOOS=linux GOARCH=amd64 go build -o ${APP_BUILD_DIR}/${APP_NAME} ../cmd/aws_api/aws_tcpdump_main.go

copy_files: compile
	cp ${APP_BUILD_DIR}/${APP_NAME} ${APP_BUILD_DIR}/usr/local/bin &&\
	cp ../cmd/aws_api/SubnetRecordingConfig.json ${APP_BUILD_DIR}/etc/${APP_NAME}
