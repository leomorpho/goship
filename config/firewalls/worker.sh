#!/bin/bash
# This firewall script allows access to the following ports:
# - 22: SSH

# Reset UFW to default settings
sudo ufw reset

# Deny all other incoming traffic
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH
sudo ufw allow ssh

# Allow internal network traffic (assuming your private network range is 10.124.0.0/24)
sudo ufw allow from 10.124.0.0/24

# Enable UFW
sudo ufw enable

# Show UFW status
sudo ufw status verbose
