#!/bin/bash
make docker-dev
docker tag ntfy:mvp1 192.168.1.101:5000/ntfy:mvp1
docker login http://192.168.1.101:5000 -u none -p none
docker push 192.168.1.101:5000/ntfy:mvp1
