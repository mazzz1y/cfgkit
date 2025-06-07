# cfgkit

A simple tool for distributing templated configs via HTTP.

## Usage

```bash
docker run -d -p 8080:8080 -v $PWD/config:/config ghcr.io/mazzz1y/cfgkit:latest
curl -u "device1:secret" 127.0.0.1:8080
```

### Example Configuration

*Config files can be split into multiple files using top-level keys.*

```yaml
devices:
  device1:
    template: default
    password: secret
    replacements:
      auth:
        username: device1
        password: password!

replacements:
  api_endpoint: http://example.com
  timeout: 30

templates:
  default:
    type: json # "raw" will disable JSON consistency checking and formatting
    data: |
      {
        "user": {{ .Device.auth | toJSON }},
        "settings": {
          "api_url": "{{ .Global.api_endpoint }}",
          "timeout": {{ .Global.timeout }}
        }
      }
```