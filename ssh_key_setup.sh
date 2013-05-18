#!/usr/bin/env bash

# Ensure that SSH_AUTH_SOCK is kept
if [ -n "$SSH_AUTH_SOCK" ]; then
    echo "SSH_AUTH_SOCK is present"
else
    echo "SSH_AUTH_SOCK is not present, adding as env_keep to /etc/sudoers"
    echo "Defaults env_keep+=\"SSH_AUTH_SOCK\"" >> "/etc/sudoers"
    mkdir -p /root/.ssh/
    chmod 700 /root/.ssh/
    echo 'rtfb.lt' | ssh-keyscan -H -f - >> /root/.ssh/known_hosts
    cp /vagrant/priv_key /root/.ssh/id_rsa
fi
