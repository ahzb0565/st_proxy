#!/bin/bash

# Set default service name
service_name=${1:-"go_proxy.service"}

# Check if go_proxy service is already registered
if systemctl list-unit-files | grep -q "$service_name"; then
    echo "$service_name is already registered. Stopping the service..."
    systemctl stop "$service_name"
    systemctl disable "$service_name"
fi

cp ./"$service_name" /etc/systemd/system/"$service_name"
systemctl daemon-reexec
systemctl daemon-reload
systemctl enable "$service_name"