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
