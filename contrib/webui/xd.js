/** xd.js -- xd json rpc api */



function XDUI()
{
    this._url = "http://127.0.0.1:1488/ecksdee/api";
}


XDUI.prototype._apicall = function(call, cb)
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
XDUI.prototype.eachTorrent = function(cb)
{
    var self = this;
    self._apicall({

    }, function(j) {
        if(j.error) {

        } else {

        }
    });
};
