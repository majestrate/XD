var XD = require("./xd.js").XD;
var UI = require("./ui.js").UI;

/** start ui on window loaded */
window.onload = function() {
    var elem = document.getElementById("xd-root");
    var path = window.location.protocol + "//"+window.location.host + "/ecksdee/api";
    var xd = new XD(path);
    var ui = new UI(xd);
    ui.build(elem);
    console.log("api url is set to "+path);
    xd.update(ui);
    var id = setInterval(function() {
        xd.update(ui);
    }, 5000);
};
