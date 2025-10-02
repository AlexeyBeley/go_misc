SHELL := /bin/bash
AWS_ACCOUNT_ID := $(shell aws sts get-caller-identity --query Account --output text)

CURRENT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR := $(realpath ${CURRENT_DIR}/../../)

BUILD_DIR := /tmp/slack_server_build

REGION := us-east-1
CLUSTER_NAME := PlaceHolder
SERVICE_NAME := PlaceHolder


test:
	echo ${AWS_ACCOUNT_ID}

init_build_dir:
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)


copy_to_build: init_build_dir
	@cp -r ${ROOT_DIR}/* ${BUILD_DIR}
	@cp  ${CURRENT_DIR}/Dockerfile ${BUILD_DIR}
	@cp /opt/human_api/slack_server_configuration.json ${BUILD_DIR}/slack_server_configuration.json


build_image: copy_to_build 
	cd ${BUILD_DIR} &&\
	docker build -t slack-server --platform linux/amd64 . 


update_artifact: build_image
	aws ecr get-login-password --region ${REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com
	docker tag slack-server:latest ${AWS_ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/slack_bot:latest
	docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/slack_bot:latest

deploy: update_artifact deploy_raw

deploy_raw:
	aws ecs update-service --region ${REGION} --cluster ${CLUSTER_NAME} --service ${SERVICE_NAME} --force-new-deployment | cat
	

