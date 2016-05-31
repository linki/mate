#!/usr/bin/env bash
set -e
set -x

IMAGE=${IMAGE:-pierone.stups.zalan.do/teapot/mate}
TAG=${TAG:-v0.0.0-SNAPSHOT}

GITHEAD=$(git rev-parse HEAD)
echo $GITHEAD
if [ -n "$(git status --porcelain)" ]; then
  GITSTATUS=$(git status --porcelain);
else
  GITSTATUS="no changes";
fi
echo "{\"url\": \"$GIT_URL\", \"revision\": \"$GITHEAD\", \"status\": \"$GITSTATUS\"}" > scm-source.json

docker build -t $IMAGE:$TAG .
