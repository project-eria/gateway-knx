# ERIA Project - Gateway for KNX

## Configuration file (gateway-owm.json)
```
port: 80
gateway:
  host: 127.0.0.1
  port: 3671
tunnelMode: true
devices:
- type: LightBasic
  name: <device full name>
  ref: <device ref, for sub url>
  states:
    on:
      grpAddr: <KNX state Group Address>
  actions:
    toggle:
      grpAddr: <KNX action Group Address>
...
```

## TODO
* Handle KNX gateway timeout
* bridge mode
* handle others dpt types