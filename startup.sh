#!/bin/sh

# 1. Provide a dummy configuration so NGINX can bind to PORT instantly.
# This prevents Render from killing the container with a timeout error
# while we wait for the Go API nodes to finish building.
export PORT=${PORT:-8080}

cat <<EOF > /etc/nginx/conf.d/default.conf
server {
    listen $PORT;
    location / {
        return 503 '{"error": "API nodes are currently compiling and deploying, please wait..."}';
        default_type application/json;
    }
}
EOF

# 2. Start NGINX in the background to satisfy the port health check immediately
nginx -g "daemon off;" &
NGINX_PID=$!
echo "NGINX started with dummy config on port $PORT. Satisfying Render healthcheck."

# 3. Wait indefinitely for the API nodes to finish their build/deploy process and register in DNS
resolve_ip() {
  local host=$1
  local ip=""
  while [ -z "$ip" ]; do
    echo "Waiting for $host to finish building and resolve via ping..." >&2
    ip=$(ping -c 1 $host 2>/dev/null | awk -F'[()]' '/PING/{print $2}')
    if [ -z "$ip" ]; then
      sleep 5
    fi
  done
  echo "$ip"
}

export API1_IP=$(resolve_ip api1)
export API2_IP=$(resolve_ip api2)
export API3_IP=$(resolve_ip api3)

echo "Success! Resolved IPs: api1=$API1_IP, api2=$API2_IP, api3=$API3_IP"

# 4. Generate the real NGINX upstream config using the raw IPs
envsubst '${API1_IP} ${API2_IP} ${API3_IP} ${PORT}' < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf

# 5. Reload NGINX to apply the load balancing config
echo "Reloading NGINX with real upstream config..."
nginx -s reload

# 6. Keep the container running
wait $NGINX_PID
