# krakend-spew
Spew exporter middleware for the KrakenD framework

**Master Caution: running this module in production will kill your performance!!!**

## Usage

Add the `ProxyFactory`, `BackendFactory`, `ClientFactory` and/or `RunServer` functions in your factory stack as showed in the example:

```
// spew http client factory wrapper
cf := spew.ClientFactory(logger, proxy.NewHTTPClient, *output)
// spew backend proxy wrapper
bf := spew.BackendFactory(logger, proxy.CustomHTTPProxyFactory(cf), *output)
// spew proxy wrapper
pf := spew.ProxyFactory(logger, proxy.NewDefaultFactory(bf, logger), *output)
// spew router wrapper
runServer := spew.RunServer(logger, router.RunServer, *output)

routerFactory := krakendgin.NewFactory(krakendgin.Config{
	Engine:         gin.Default(),
	ProxyFactory:   pf,
	Logger:         logger,
	HandlerFactory: krakendgin.EndpointHandler,
	RunServer:      krakendgin.RunServerFunc(runServer),
})

routerFactory.NewWithContext(ctx).Run(serviceConfig)
```	

Build and start the example:

```
$ go build -o sample ./example
$ ./sample -c example/krakend.json -d -l DEBUG
[GIN-debug] [WARNING] Now Gin requires Go 1.6 or later and Go 1.7 will be required soon.

[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

{{0xc000216480 [] 0x1555a60 0x1564500 {0 [KRAKEND]} 0x1564e40} 0xc0001bc0c0 0x1564e40}
[KRAKEND] DEBUG: Debug enabled
[GIN-debug] GET    /__debug/*param           --> github.com/devopsfaith/krakend-spew/vendor/github.com/devopsfaith/krakend/router/gin.DebugHandler.func1 (3 handlers)
[GIN-debug] POST   /__debug/*param           --> github.com/devopsfaith/krakend-spew/vendor/github.com/devopsfaith/krakend/router/gin.DebugHandler.func1 (3 handlers)
[GIN-debug] PUT    /__debug/*param           --> github.com/devopsfaith/krakend-spew/vendor/github.com/devopsfaith/krakend/router/gin.DebugHandler.func1 (3 handlers)
[GIN-debug] GET    /nick/:nick               --> github.com/devopsfaith/krakend-spew/vendor/github.com/devopsfaith/krakend/router/gin.CustomErrorEndpointHandler.func1 (3 handlers)
[GIN-debug] GET    /combination/:id          --> github.com/devopsfaith/krakend-spew/vendor/github.com/devopsfaith/krakend/router/gin.CustomErrorEndpointHandler.func1 (3 handlers)
```

After sending a test request to http://localhost:8000/nick/kpacha, you will see some log lines

```
[KRAKEND] DEBUG: spew: proxy request captured: proxy_L25pY2svOm5pY2s=
[KRAKEND] DEBUG: spew: proxy request captured: backend_L3VzZXJzL3t7Lk5pY2t9fQ==
[KRAKEND] DEBUG: spew: proxy request captured: backend_LzIuMC91c2Vycy97ey5OaWNrfX0=
[KRAKEND] DEBUG: spew: capturing http request and response at the backend layer
[KRAKEND] DEBUG: spew: capturing http request and response at the backend layer
[KRAKEND] DEBUG: spew: proxy executed
[KRAKEND] DEBUG: spew: proxy response captured: backend_LzIuMC91c2Vycy97ey5OaWNrfX0=
[KRAKEND] DEBUG: spew: capturing http request and response at the backend layer
[KRAKEND] DEBUG: spew: proxy executed
[KRAKEND] DEBUG: spew: proxy response captured: backend_L3VzZXJzL3t7Lk5pY2t9fQ==
[KRAKEND] DEBUG: spew: proxy executed
[KRAKEND] DEBUG: spew: proxy response captured: proxy_L25pY2svOm5pY2s=
[GIN] 2018/09/25 - 19:12:27 | 200 |   692.97048ms |             ::1 | GET      /nick/kpacha
[KRAKEND] DEBUG: spew: capturing http request and response at the router layer
...
```

You can stop the KrakenD and check the `.txt` files generated by the module:

```
2,0K 25 sep 19:12 backend_L3VzZXJzL3t7Lk5pY2t9fQ==_1537895547814979000.txt
1,8K 25 sep 19:12 backend_LzIuMC91c2Vycy97ey5OaWNrfX0=_1537895547800941000.txt
 92K 25 sep 19:12 client_aHR0cHM6Ly9hcGkuYml0YnVja2V0Lm9yZy8yLjAvdXNlcnMva3BhY2hh_1537895547798571000.txt
 92K 25 sep 19:12 client_aHR0cHM6Ly9hcGkuYml0YnVja2V0Lm9yZy8yLjAvdXNlcnMva3BhY2hh_1537895547800824000.txt
104K 25 sep 19:12 client_aHR0cHM6Ly9hcGkuZ2l0aHViLmNvbS91c2Vycy9rcGFjaGE=_1537895547814647000.txt
1,9K 25 sep 19:12 client_basic_aHR0cHM6Ly9hcGkuYml0YnVja2V0Lm9yZy8yLjAvdXNlcnMva3BhY2hh_1537895547796264000.txt
1,9K 25 sep 19:12 client_basic_aHR0cHM6Ly9hcGkuYml0YnVja2V0Lm9yZy8yLjAvdXNlcnMva3BhY2hh_1537895547798755000.txt
2,7K 25 sep 19:12 client_basic_aHR0cHM6Ly9hcGkuZ2l0aHViLmNvbS91c2Vycy9rcGFjaGE=_1537895547812621000.txt
2,3K 25 sep 19:12 proxy_L25pY2svOm5pY2s=_1537895547815244000.txt
 66K 25 sep 19:12 router_L25pY2sva3BhY2hh_1537895547816402000.txt
```