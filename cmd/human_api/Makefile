SHELL := /bin/bash

CURRENT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR := $(realpath ${CURRENT_DIR}/../../)

BUILD_DIR := /tmp/slack_server_build

test:
	echo ${ROOT_DIR}

init_build_dir:
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)


copy_to_build: init_build_dir
	@cp -r ${ROOT_DIR}/* ${BUILD_DIR}
	@cp  ${CURRENT_DIR}/Dockerfile ${BUILD_DIR}


build_image: copy_to_build 
	cd ${BUILD_DIR} &&\
	docker build -t slack-server . 
