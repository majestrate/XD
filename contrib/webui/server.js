var docroot = "docroot";

var express = require("express");
var app = express();
app.use("/", express.static(docroot));
app.listen(8000, function() { console.log("listening on 0.0.0.0:8000")});
