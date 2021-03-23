#!/usr/bin/env bash

version="$1"
git clean -xdf
GIT_VERSION="" GOOS=windows GOARCH=386 make clean build && mv XD.exe XD-$version-win32.exe && gpg --sign --detach XD-$version-win32.exe
GIT_VERSION="" GOOS=windows GOARCH=amd64 make clean build && mv XD.exe XD-$version-win64.exe && gpg --sign --detach XD-$version-win64.exe
GIT_VERSION="" GOOS=darwin GOARCH=amd64 make clean build && mv XD XD-$version-darwin && gpg --sign --detach XD-$version-darwin
GIT_VERSION="" GOOS=linux GOARCH=386 make clean build && mv XD XD-$version-linux-i386 && gpg --sign --detach XD-$version-linux-i386
GIT_VERSION="" GOOS=linux GOARCH=amd64 make clean build && mv XD XD-$version-linux-amd64 && gpg --sign --detach XD-$version-linux-amd64
GIT_VERSION="" GOOS=linux GOARCH=arm make clean build && mv XD XD-$version-linux-arm && gpg --sign --detach XD-$version-linux-arm
GIT_VERSION="" GOOS=linux GOARCH=arm GOARM=6 make clean build && mv XD XD-$version-linux-rpi && gpg --sign --detach XD-$version-linux-rpi
GIT_VERSION="" GOOS=linux GOARCH=ppc64 make clean build && mv XD XD-$version-linux-ppc64 && gpg --sign --detach XD-$version-linux-ppc64
GIT_VERSION="" GOOS=freebsd GOARCH=amd64 make clean build  && mv XD XD-$version-freebsd-amd64 && gpg --sign --detach XD-$version-freebsd-amd64
