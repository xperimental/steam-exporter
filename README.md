# steam-exporter

A simple exporter for getting server information from game-servers which are compatible with
Valve's [A2S_INFO Query](https://developer.valvesoftware.com/wiki/Server_queries#A2S_INFO).

## Installation

### Docker Image

The easiest installation method is using the provided Docker image:

```plain
docker pull ghcr.io/xperimental/steam-exporter:latest
```

The tag `latest` is built from the current version in `master`, released versions are available with a separate tag.

### From source

If you have a recent Go installation and GNU Make on your system, building the exporter should be as easy as cloning the
repository and then running `make`:

```bash
git clone https://github.com/xperimental/steam-exporter.git
cd steam-exporter
make
```

## Usage

The exporter takes a single command-line parameter specifying the location of the configuration file:

```plain
$ steam-exporter --help
Usage of ./steam-exporter:
  -c, --config-file string   Path to configuration file. (default "steam-exporter.yml")
  -v, --verbose              Show debugging output.
```

The configuration file is a YAML file. The simplest form is just a list of server addresses which should be scraped for
metrics:

```yaml
servers:
- address: 127.0.0.1:65000
```

### Exported metrics

| Name | Description |
| ---- | ----------- |
| `steam_server_up` | Set to 1 if the server is reachable. |
| `steam_server_response_time_seconds` | Shows the response time of the server in seconds. |
| `steam_server_players_total` | Shows current number of players on the server. |
| `steam_server_max_players_total` | Shows current number of players on the server. |
| `steam_server_bots_total` | Shows current number of players on the server. |

All metrics have an `address` label listing the address used to scrape data from the server.
