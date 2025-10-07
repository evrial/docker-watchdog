1. Install: 
```sh
go build
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

Example usage
```sh
# Default run
./docker-watchdog

# 5s timeout, 60s cooldown
./docker-watchdog -timeout 5 -cooldown 60
```
