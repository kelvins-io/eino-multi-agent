#!/bin/sh
set -eu

base="${VITE_BASE_PATH:-/}"
base=$(printf '%s' "$base" | sed 's:/*$::')
if [ -z "$base" ] || [ "$base" = "/" ]; then
  exit 0
fi
case "$base" in
  /*) ;;
  *) base="/$base" ;;
esac

cat > /etc/nginx/conf.d/default.conf <<EOF
server {
    listen 80;
    server_name _;
    client_max_body_size 64m;

    gzip on;
    gzip_types text/css application/javascript application/json image/svg+xml;

    location = ${base} {
        return 301 ${base}/;
    }

    location ${base}/api/ {
        resolver 127.0.0.11 valid=10s ipv6=off;
        set \$api_host api;
        rewrite ^${base}(/api/.*)\$ \$1 break;
        proxy_pass http://\$api_host:8180;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    location ${base}/ {
        alias /usr/share/nginx/html/;
        try_files \$uri \$uri/ @spa;
    }

    location @spa {
        root /usr/share/nginx/html;
        rewrite ^ /index.html break;
    }
}
EOF
