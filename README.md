# Dynamic DNS Service
Simple dynamic DNS service for A and AAAA records. The client updates the IP record for the specified domain.

**Important:** the api service must run behind a secure HTTPs proxy.

## Server

### Server Config
```yaml
ttl: 2m
keys:
    foo.sample.com: "RANDOM_60_CHAR_KEY_HERE"
```

### Systemd Service Sample with Traefik Proxy
```ini
[Unit]
Description=DynDNS Service
After=docker.service
Requires=docker.service

[Install]
WantedBy=multi-user.target

[Service]
TimeoutStartSec=0
Restart=always
ExecStartPre=-/usr/bin/docker kill ddns
ExecStartPre=-/usr/bin/docker rm ddns
ExecStartPre=-/usr/bin/docker pull desertbit/ddns
ExecStart=/usr/bin/docker run \
  --name ddns \
  -p 53:53/udp \
  -p 53:53/tcp \
  --label "traefik.enable=true" \
  --label "traefik.http.routers.ddns.rule=Host(`ns.sample.com`)" \
  --label "traefik.http.services.ddns.loadbalancer.server.port=80" \
  --label "traefik.http.routers.ddns.entrypoints=websecure" \
  -v /data/ddns:/data \
  --network="http_network" \
  desertbit/ddns server \
    --conf /data/ddns.yaml \
    --db /data/ddns.db
ExecStop=/usr/bin/docker stop ddns
```

## Client
```ini
[Unit]
Description=DynDNS Service
After=docker.service
Requires=docker.service

[Install]
WantedBy=multi-user.target

[Service]
TimeoutStartSec=0
Restart=always
ExecStartPre=-/usr/bin/docker kill ddns
ExecStartPre=-/usr/bin/docker rm ddns
ExecStartPre=-/usr/bin/docker pull desertbit/ddns
ExecStart=/usr/bin/docker run \
  --name ddns \
  desertbit/ddns client \
    --url "https://ns.sample.com" \
    --domain "foo.sample.com" \
    --interval 1m \
    --key "RANDOM_60_CHAR_KEY_HERE"
ExecStop=/usr/bin/docker stop ddns
```

## DNS Setup

Given your domain is sample.com, log in to your domain hoster and create a subdomain with a dedicated NS record that points to your newly created DNS. The NS record itself should point to a A record that is also created, which again points to the IP address of your DNS, like this:

```
foo                      IN NS      ns
ns                       IN A       <put ipv4 of dns server here>
ns                       IN AAAA    <optional, put ipv6 of dns server here>
```

## SystemD ResolveD Port Issue
systemd-resolved uses port 53. Instead of disabling this service, remap incoming
packets from wan to a new port. The docker container will listen on this new port.
https://github.com/dprandzioch/docker-ddns/issues/5

```
table ip router {
    # Both need to be set even when one is empty.
    chain prerouting {
        type nat hook prerouting priority 0;
        iifname $wan_iface udp dport 53 counter redirect to :55553
    }
    chain postrouting {
        type nat hook postrouting priority 100;
    }
}
```

## Credits
- http://mkaczanowski.com/golang-build-dynamic-dns-service-go
- https://www.davd.io/build-your-own-dynamic-dns-in-5-minutes/
- https://github.com/muka/ddns
- https://serverfault.com/questions/804492/dns-servfail-at-some-of-nameservers/804526

## License
MIT License
