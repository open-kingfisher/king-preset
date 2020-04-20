#!/bin/bash

CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')
echo $CA_BUNDLE

