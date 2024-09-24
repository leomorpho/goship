#!/bin/bash
# This firewall script allows access to the following ports:
# - 6379: Redis
# - 8080: AsyncMon
# - 22: SSH

YOUR_LAPTOP_IP="192.000.0.000" # SET YOUR LAPTOP IP

# Reset UFW to default settings
sudo ufw reset

# Deny all other incoming traffic
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH
sudo ufw allow ssh

# Allow internal network traffic (assuming your private network range is 10.124.0.0/24)
sudo ufw allow from 10.124.0.0/24

# Allow Redis port (assuming Redis is exposed to internal network only)
sudo ufw allow from 10.124.0.0/24 to any port 6379

# Allow AsyncMon port for internal network
sudo ufw allow from 10.124.0.0/24 to any port 8080

# Allow AsyncMon port for your laptop IP
sudo ufw allow from $YOUR_LAPTOP_IP to any port 8080

# Enable UFW
sudo ufw enable

# Show UFW status
sudo ufw status verbose
