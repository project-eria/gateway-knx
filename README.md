# ERIA Project - Gateway for KNX

## Configuration file (gateway-owm.json)
````
{
  "GWXaalAddr": "<gateway xAAL address>",
  "GatewayIP": "<KNX gateway IP",
  "GatewayPort": 3671,
  "TunnelMode": true,
  "Devices": [
    {
      "Type": "lamp.basic",
      "Name": "Lampe Etage",
      "XaalAddr": "<xAAL address>",
      "Groups": [
        {
          "GrpAddr": "1/1/160",
          "DPT": "DPT_1001",
          "Attribute": "light"
        }
      ]
    }
  ]
}
````

## TODO
* Handle KNX gateway timeout
* bridge mode
* handle others dpt types