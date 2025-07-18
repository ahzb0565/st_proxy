#!/bin/bash

service_name=${1:-"go_proxy.service"}

systemctl status "$service_name"

journalctl -u -n 100 "$service_name"