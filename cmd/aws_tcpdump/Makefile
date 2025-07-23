
SHELL := /bin/bash

CUR_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
ROOT_DIR:=  $(realpath ${CUR_DIR}/../../)

GOENVPATH:=$(go env GOPATH)

BUILD_DIR:=${ROOT_DIR}/_build

tmp:
	echo ${ROOT_DIR}

build_dir: 
	mkdir -p ${BUILD_DIR}

build: build_dir 
	go build -o ${BUILD_DIR}/aws_tcpdump

install_data:
	mkdir -p /opt/aws_api/tcpdump/data/IamDataDir &&\
	mkdir -p /opt/aws_api/tcpdump/output &&\
	cp ${ROOT_DIR}/aws_api/data/template* /opt/aws_api/data/IamDataDir &&\
	cp ${ROOT_DIR}/cmd/aws_api/tcpdump/tcpdump.conf /opt/aws_api/tcpdump/data/
	

install: install_data build
	mv ${BUILD_DIR}/aws_tcpdump /usr/local/bin/