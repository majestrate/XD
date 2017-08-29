(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
var XD = require("./xd.js");

},{"./xd.js":2}],2:[function(require,module,exports){
/** xd.js -- xd json rpc api */



function XDAPI()
{
    this._url = "http://127.0.0.1:1488/ecksdee/api";
}


XDAPI.prototype._apicall = function(call, cb)
{
    var self = this;
    $.ajax({
        url: self._url,
        contentType: "text/json; charset=UTF-8",
        content: JSON.stringify(call),
        success: function(any, text, xhr) {
            var j = JSON.parse(text);
            console.log(call, j);
            cb(j);
        }
    });
};

/** fetch a list of torrents and call a callback on each fetched */
XDAPI.prototype.eachTorrent = function(cb)
{
    var self = this;
    self._apicall({

    }, function(j) {
        if(j.error) {

        } else {

        }
    });
};


module.exports = {
    "XD": XD
};

},{}]},{},[1]);
