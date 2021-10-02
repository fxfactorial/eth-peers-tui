Need to build off this branch as your geth node

https://github.com/fxfactorial/go-ethereum/tree/p2p-chans

run that node, it has the websocket opening at 8080

then run this cli via the usual `go build`

# ip location db

get it via here: https://dev.maxmind.com/geoip/geolite2-free-geolocation-data?lang=en

run with `mmdb` flag to specify an ip location db, otherwise I'm assuming GeoLite2-City_20210928/