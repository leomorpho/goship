#!/bin/bash
# This firewall script allows access to the following ports:  
# - 443: HTTPS
# - 22: SSH

# Enable UFW and set default rules
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw enable

# Allow SSH
sudo ufw allow ssh

# Fetch Cloudflare IPs and allow them for HTTPS
echo "Fetching Cloudflare IPs..."
curl -s https://www.cloudflare.com/ips-v4/ -o cloudflare-ips-v4.txt

echo "Configuring UFW to allow Cloudflare IPs for HTTPS..."
while IFS= read -r ip; do
  sudo ufw allow from $ip to any port 443
done < cloudflare-ips-v4.txt

# Clean up
rm cloudflare-ips-v4.txt

# Enable UFW
sudo ufw enable

echo "UFW configuration completed successfully."

###########################
# Fail-to-ban
###########################
# Setup fail to ban
apt update && apt upgrade
apt install fail2ban -y
# Verify it's installed
fail2ban-client -h