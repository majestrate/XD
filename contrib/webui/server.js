var docroot = "docroot";
var backend = "http://127.0.0.1:1488";


var express = require("express");
var httpProxy = require("http-proxy");
var apiProxy = httpProxy.createProxyServer();

var app = express();
app.post("/ecksdee/api", function(req, res) {
    apiProxy.web(req, res, {target: backend});
});
app.use("/", express.static(docroot));

app.listen(8000, function() { console.log("listening on 0.0.0.0:8000")});
