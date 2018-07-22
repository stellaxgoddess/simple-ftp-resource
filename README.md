[![Docker Build Status](https://img.shields.io/docker/build/xperimental/simple-ftp-resource.svg?style=flat-square)](https://hub.docker.com/r/xperimental/simple-ftp-resource/)

# simple-ftp-resource

This provides a Concourse resource which can be used for interacting with an FTP server.

Currently only upload (`put`) is supported.

## Usage

1. Define a new `resource_type`
2. Define a resource using the resource type
3. Add a `put` step detailing what to upload

```yaml
resource_types:
- name: ftp
  type: docker-image
  source:
    repository: xperimental/simple-ftp-resource

resources:
- name: server
  type: ftp
  source:
    host: ftp.example.de:21
    user: username
    password: password
    tls: true

jobs:
- name: upload
  plan:
  - put: site
    resource: server
    params:
      local: website/
      remote: /
```