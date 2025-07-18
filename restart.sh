#!/bin/bash

service_name=${1:-"go_proxy.service"}

systemctl restart "$service_name"