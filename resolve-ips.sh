#!/bin/sh

resolve_ip() {
  local host=$1
  local ip=""
  while [ -z "$ip" ]; do
    echo "Waiting for $host to resolve via ping..." >&2
    # ping in Alpine outputs: PING api1 (10.0.0.5): 56 data bytes
    # We use awk to extract the IP inside the parentheses
    ip=$(ping -c 1 $host 2>/dev/null | awk -F'[()]' '/PING/{print $2}')
    if [ -z "$ip" ]; then
      sleep 2
    fi
  done
  echo "$ip"
}

echo "Resolving API backend IPs to bypass NGINX DNS limitations..."

export API1_IP=$(resolve_ip api1)
export API2_IP=$(resolve_ip api2)
export API3_IP=$(resolve_ip api3)

echo "Successfully resolved IPs: api1=$API1_IP, api2=$API2_IP, api3=$API3_IP"

# Ensure PORT is set (Render injects it automatically, but fallback for local Docker)
export PORT=${PORT:-8080}

echo "Injecting variables into NGINX config..."
envsubst '${API1_IP} ${API2_IP} ${API3_IP} ${PORT}' < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf

echo "Generated /etc/nginx/conf.d/default.conf:"
cat /etc/nginx/conf.d/default.conf
