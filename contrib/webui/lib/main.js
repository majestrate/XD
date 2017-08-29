var XD = require("./xd.js").XD;
var UI = require("./ui.js").UI;

/** start ui on window loaded */
window.onload = function() {
    var elem = document.getElementById("xd-root");
    var ui = new UI();
    ui.build(elem);
    var xd = new XD();
    var id = setInterval(function() {
        xd.update(ui);
    }, 1000);
};
