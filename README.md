# LXC-UI-API
LXC-UI-API is a backend API that allows lxc to be used on LXD-UI and INCUS-UI.

# How to use
1. Create config. yaml in the same level directory of the program and fill in the content:
```yaml
server:
  ip: "0.0.0.0"
  port: 8443
  cert: "incus-ui.crt"          # Client certificate
  server-cert: "server.crt"     # If empty, automatically generated will be used
  server-cert-key: "server.key" # If empty, automatically generated will be used
```
2. Extract the ui folder from LXD-UI or INCUS-UI to the program directory.\
   You can also obtain it from https://github.com/cmspam/incus-ui.
   Or use LXC_UI:
```
LXC_UI=/opt/incus/ui ./lxc-ui-api
```
3. Run it! You will see:
```
./lxc-ui-api
Start LXC-API service: 0.0.0.0:8443
Request Method: GET | Request API: /1.0
Request Method: GET | Request API: /1.0/projects/default
Request Method: GET | Request API: /1.0/projects
Request Method: GET | Request API: /1.0/operations
Request Method: GET | Request API: /1.0/certificates
```

# API Support List
