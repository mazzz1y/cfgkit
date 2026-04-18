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

A template body can also be loaded from a file on disk instead of inlined:

```yaml
templates:
  default:
    type: json
    file: ./templates/default.json
```

Variables are resolved sequentially in declared order before the main template renders. Device variables are resolved first, then global variables, so global variables can reference `.Device.*` values. Each variable can reference any previously-resolved variable via `.Device.*` or `.Global.*`, including earlier siblings inside the same nested map. Because every variable is fully resolved before template execution, the result of a function-style variable can be piped into `fromJson` and used as structured data:

```yaml
variables:
  items:
    tags: |
      {{- list "alpha" "beta" "gamma" | mustToJson -}}
  payload: |
    {{- $entries := list -}}
    {{- $entries = append $entries (dict
      "labels" (.Global.items.tags | fromJson)
      "enabled" true
    ) -}}
    {{- $entries | mustToJson -}}
```

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

#### External Validation

Templates support an optional `check` field that runs an external command to validate the rendered output before sending the response. The rendered config is written to a temporary file inside a unique directory. The following template variables are available:

* `{{ .TemplateFilePath }}` — full path to the temporary config file
* `{{ .TemplateFileDir }}` — the directory containing the temporary file

The temporary directory is removed after the check completes.

Exec form (no shell):

```yaml
templates:
  default:
    type: json
    check: ["mycheck", "--validate", "{{ .TemplateFilePath }}"]
    data: |
      { ... }
```

Shell form (wrapped in `sh -c`):

```yaml
templates:
  default:
    type: json
    check: "mycheck --validate {{ .TemplateFilePath }}"
    data: |
      { ... }
```

If the command exits with a non-zero status, the request returns a 500 error with the command's stderr. The check command has a 5-second timeout.