
SHELL := /bin/bash

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

GOENVPATH:=$(go env GOPATH)

BUILD_DIR:=${ROOT_DIR}/_build

build_dir: 
	mkdir -p ${BUILD_DIR}

build: build_dir 
	go build -o ${BUILD_DIR}/fapono

install: build
	mv ${BUILD_DIR}/fapono /usr/local/bin/