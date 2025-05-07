
SHELL := /bin/bash

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BUILD_DIR:= ${ROOT_DIR}/build
BUILD_TMP_DIR:= ${BUILD_DIR}/_build

APP_NAME := aws_api
APP_BUILD_DIR:=${BUILD_TMP_DIR}/${APP_NAME}

intall-golang:
	curl https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
	rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz
	export PATH=$PATH:/usr/local/go/bin
	 $HOME/.profile.

install-requirements intall-golang:
	sudo apt update
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


compile: init_build_dir install-requirements
	GOOS=linux GOARCH=amd64 go build -o ${APP_BUILD_DIR}/${APP_NAME} ../cmd/aws_api/aws_tcpdump_main.go

copy_files: compile
	cp ${APP_BUILD_DIR}/${APP_NAME} ${APP_BUILD_DIR}/usr/local/bin &&\
	cp ../data ${APP_BUILD_DIR}/usr/local/bin






