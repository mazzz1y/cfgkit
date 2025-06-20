# cfgkit

A simple tool for distributing templated configs via HTTP.

## Usage

```bash
docker run -d -p 8080:8080 -v $PWD/config:/config ghcr.io/mazzz1y/cfgkit:latest
curl -u "device1:secret" 127.0.0.1:8080
```

### Configuration

#### Basic Example

```yaml
devices:
  device1:
    template: default
    password: secret
    variables:
      auth:
        username: device1
        password: password!

variables:
  api_endpoint: http://example.com
  timeout: 30

templates:
  default:
    type: json # "raw" will disable JSON consistency checking and formatting
    data: |
      {
        "auth": {{ .Device.auth | toJson }},
        "settings": {
          "api_url": "{{ .Global.api_endpoint }}",
          "timeout": {{ .Global.timeout }}
        }
      }
```

Templates and variables are resolved recursively, so each type of variable can be used anywhere. For example, "global" variables can be used in "device" variables and vice versa.
Configuration files can be split into multiple files using top-level keys.

#### Available functions:

* `{{ fromFile "path/to/file.txt" }}` — Reads any text file from the filesystem into a string.
* `{{ .String | fromYaml }}` — Converts YAML string into struct.
* All [Sprig](https://masterminds.github.io/sprig/) functions.
* All built-in Golang template functions and advanced YAML features, such as anchors.

This allows you to leverage the full power of YAML. It's dirty and stupid, but very powerful.

For example, you can use pseudo-functions like the one below to get a user's password from a third-party config:

```
variables:
  functions:
    password: |
      {{- userName := .Device.Name }}
      {{- range $users := (fromFile "/config/server.json" | fromJson).users }}
        {{- if eq $user.name $userName -}}
          {{- $user.password -}}
        {{- end -}}
      {{- end -}}
```