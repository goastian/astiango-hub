#!/bin/bash

# Fail on error
set -e

# Ensure directory mode of /tmp is world-writable
chmod 777 /tmp

# Update package index
apt-get update

# Install essential dependencies
apt-get install -y \
    curl \
    wget \
    zip \
    unzip \
    git \
    iputils-ping \
    nginx \
    jq \
    net-tools \
    fonts-wqy-zenhei \
    fonts-noto-cjk \
    fontconfig \
    locales

# Generate Chinese locale
locale-gen zh_CN.UTF-8
update-locale

# Add source /etc/profile to ~/.bashrc
echo "source /etc/profile" >> ~/.bashrc