#!/usr/bin/env bash

kubectl delete -f deployment_all_in_one.yaml
kubectl delete secret king-preset -n kingfisher