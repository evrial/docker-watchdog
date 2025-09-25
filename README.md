1. Install: 
```sh
go install
```
2. Install [uv](https://docs.astral.sh/uv/).
3. Install [Apprise](https://github.com/caronc/apprise): 
```sh
uv tool install apprise
```
4. Configure Apprise, one service per line: 
```sh
echo "tgram://bottoken/ChatID" > ~/.apprise
```
5. Create systemd service `/etc/systemd/system/docker-watchdog.service`
```
[Unit]
Description=Docker Watchdog
After=network-online.target docker.service
Wants=network-online.target

[Service]
Environment=PATH=/home/dietpi/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ExecStart=/usr/local/bin/docker-watchdog
User=dietpi
Restart=always
RestartSec=2
Nice=-10
LimitRTPRIO=95
LimitMEMLOCK=infinity

[Install]
WantedBy=multi-user.target
```

Example usage
```sh
# Default run
./docker-watchdog

# Custom log path, 5s timeout, 60s cooldown
./docker-watchdog -log /tmp/watchdog.log -timeout 5 -cooldown 60

# Or via environment variables
WATCHDOG_LOG=/tmp/watchdog.log WATCHDOG_TIMEOUT=5 WATCHDOG_COOLDOWN=60 ./docker-watchdog
```
