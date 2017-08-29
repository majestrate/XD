var XD = require("./xd.js").XD;
var UI = require("./ui.js").UI;

/** start ui on window loaded */
window.onload = function() {
    var elem = document.getElementById("xd-root");
    var ui = new UI();
    ui.build(elem);
    var path = window.location.protocol + "//"+window.location.host + "/ecksdee/api";
    console.log("api url is set to "+path);
    var xd = new XD(path);
    xd.update(ui);
    var id = setInterval(function() {
        xd.update(ui);
    }, 5000);
};
