# Configuration

LiaCheckScanner reads its configuration from `config/config.json`. If the file does not exist on startup, a default configuration is generated automatically.

## Configuration file location

```
config/config.json
```

You can edit this file directly or use the **Configuration** tab in the GUI.

## Full reference

```json
{
  "app_name": "LiaCheckScanner",
  "version": "1.0.0",
  "owner": "LIA - mo0ogly@proton.me",
  "theme": "dark",
  "language": "fr",
  "log_level": "INFO",
  "max_log_size": 10,
  "log_backups": 5,
  "database": {
    "repo_url": "https://github.com/MDMCK10/internet-scanners",
    "local_path": "./data/repository",
    "results_dir": "./results",
    "logs_dir": "./logs",
    "api_key": "",
    "enable_api": false,
    "api_throttle": 1.0,
    "parallelism": 4,
    "registries": ["arin", "ripe", "apnic", "lacnic", "afrinic"],
    "auto_update": false,
    "update_interval": 24
  }
}
```

## Field descriptions

### Top-level fields

| Field          | Type   | Default              | Description                                                              |
|----------------|--------|----------------------|--------------------------------------------------------------------------|
| `app_name`     | string | `"LiaCheckScanner"`  | Display name of the application.                                         |
| `version`      | string | `"1.0.0"`            | Application version string.                                             |
| `owner`        | string | `"LIA - mo0ogly@proton.me"` | Author and contact information.                                  |
| `theme`        | string | `"dark"`             | GUI theme. Accepted values: `"dark"`, `"light"`.                         |
| `language`     | string | `"fr"`               | UI language code (e.g. `"fr"`, `"en"`).                                  |
| `log_level`    | string | `"INFO"`             | Minimum log level. One of `"DEBUG"`, `"INFO"`, `"WARNING"`, `"ERROR"`, `"CRITICAL"`. |
| `max_log_size` | int    | `10`                 | Maximum size of a single log file in megabytes before rotation occurs.   |
| `log_backups`  | int    | `5`                  | Number of rotated log files to keep.                                     |

### `database` section

| Field             | Type     | Default                                              | Description                                                                                     |
|-------------------|----------|------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| `repo_url`        | string   | `"https://github.com/MDMCK10/internet-scanners"`    | URL of the Git repository containing `.nft` scanner files.                                      |
| `local_path`      | string   | `"./data/repository"`                                | Local directory where the repository is cloned.                                                 |
| `results_dir`     | string   | `"./results"`                                        | Directory for CSV export files.                                                                 |
| `logs_dir`        | string   | `"./logs"`                                           | Directory for log files.                                                                        |
| `api_key`         | string   | `""`                                                 | Optional API key (reserved for future use).                                                     |
| `enable_api`      | bool     | `false`                                              | Whether the optional API endpoint is enabled.                                                   |
| `api_throttle`    | float64  | `1.0`                                                | Delay in **seconds** between RDAP/geolocation API requests. Controls rate limiting.             |
| `parallelism`     | int      | `4`                                                  | Number of concurrent worker goroutines for RDAP enrichment.                                     |
| `registries`      | []string | `["arin","ripe","apnic","lacnic","afrinic"]`         | List of RDAP registries to query. Removing entries skips those registries during enrichment.     |
| `auto_update`     | bool     | `false`                                              | Whether to automatically pull the scanner repository on startup.                                |
| `update_interval` | int      | `24`                                                 | Interval in **hours** between automatic repository updates (only relevant if `auto_update` is true). |

## Notes on throttling and parallelism

The `api_throttle` value controls the minimum delay between successive API calls within each worker. Combined with `parallelism`, the effective maximum request rate is:

```
max requests/sec = parallelism / api_throttle
```

For example, with `parallelism: 4` and `api_throttle: 1.0`, the application makes at most 4 requests per second across all workers.

!!! warning
    Setting `api_throttle` to `0` removes all rate limiting. Some RDAP endpoints and the ip-api.com geolocation service enforce their own limits and may return errors or ban your IP if you send requests too quickly.

## Modifying configuration at runtime

Changes made in the **Configuration** tab of the GUI are written to `config/config.json` immediately when you press **Save Configuration**. The new values take effect for subsequent operations without restarting the application.

## Cache and progress files

In addition to `config/config.json`, the application maintains two data files under `build/data/`:

| File                    | Purpose                                                        |
|-------------------------|----------------------------------------------------------------|
| `rdap_cache.json`       | Caches RDAP and geolocation results keyed by IP address.       |
| `rdap_progress.json`    | Tracks progress of bulk RDAP enrichment for resume support.    |

These files are managed automatically. Deleting `rdap_cache.json` forces fresh lookups; deleting `rdap_progress.json` resets enrichment progress.
