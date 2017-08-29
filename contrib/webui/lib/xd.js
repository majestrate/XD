/** xd.js -- xd json rpc api */

var $ = require("jquery");

function XDAPI(url)
{
    this._url = url;
}


XDAPI.prototype._apicall = function(call, cb)
{
    var self = this;
    $.ajax({
        method: "POST",
        url: self._url,
        contentType: "text/json; charset=UTF-8",
        data: JSON.stringify(call),
        success: function(j, text, xhr) {
            console.log(call, j);
            cb(j);
        }
    });
};

/** get torrent information by infohash as hex */
XDAPI.prototype.getTorrentInfo = function(infohash, callback)
{
    var self = this;
    self._apicall({
        method: "XD.TorrentStatus",
        infohash: infohash
    }, function(j) {
        if(j.error) {
            callback(j.error, null);
        } else {
            callback(null, j);
        }
    });
};

/** fetch a list of torrents and call a callback on each fetched */
XDAPI.prototype.eachTorrent = function(cb)
{
    var self = this;
    self._apicall({
        method: "XD.ListTorrents"
    }, function(j) {
        if(j.error) {
            console.log("eachTorrent(): "+j.error);
        } else {
            $(j.Infohashes).each(function(idx, t) {
                if(t) {
                    self.getTorrentInfo(t, function(err, info) {
                        if(err) return;
                        cb(info);
                    });
                }
            });
        }
    });
};

XDAPI.prototype.update = function(ui)
{
    var self = this;
    console.log("update ui");

    ui.beginTorrentUpdate();
    self.eachTorrent(function(t) {
        ui.ensureTorrentWithInfo(t);
    });
    ui.commitTorrentUpdate();
};

module.exports = {
    "XD": XDAPI
};
