# LXC-UI-API
LXC-UI-API is a backend API that allows lxc to be used on LXD-UI and INCUS-UI.

# How to use
1. Create config. yaml in the same level directory of the program and fill in the content:
```yaml
server:
  ip: "0.0.0.0"
  port: 8443
  server-cert: "server.crt"     # If empty, automatically generated will be used
  server-cert-key: "server.key" # If empty, automatically generated will be used

client:
  certs:                        # If empty, token only
    - cert: "incus-ui.crt"
    - cert: "lxd-ui.crt"
  tokens:                       # If empty, tls only
    # The original content of the token was:
    # {"client_name":"lxc-ui-api","fingerprint":"0ba029714a9e1e93dee8a0f960125c2ed82c05c19906ff0e254577e2361274ee","addresses":["127.0.0.1:8443","[::1]:8443"],"secret":"8ee82edf87034f4c24fb0f2472bb8ee742cbb0822c57b8ef92b63719ad3f705e","expires_at":"0001-01-01T00:00:00Z"}
    # Encoded using base64
    - token: "eyJjbGllbnRfbmFtZSI6Imx4Yy11aS1hcGkiLCJmaW5nZXJwcmludCI6IjBiYTAyOTcxNGE5ZTFlOTNkZWU4YTBmOTYwMTI1YzJlZDgyYzA1YzE5OTA2ZmYwZTI1NDU3N2UyMzYxMjc0ZWUiLCJhZGRyZXNzZXMiOlsiMTI3LjAuMC4xOjg0NDMiLCJbOjoxXTo4NDQzIl0sInNlY3JldCI6IjhlZTgyZWRmODcwMzRmNGMyNGZiMGYyNDcyYmI4ZWU3NDJjYmIwODIyYzU3YjhlZjkyYjYzNzE5YWQzZjcwNWUiLCJleHBpcmVzX2F0IjoiMDAwMS0wMS0wMVQwMDowMDowMFoifQ=="
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
