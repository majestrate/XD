/** ui.js -- ui builder */
var util = require("./util.js");
var $ = require("jquery");

var elem = function(name, css)
{
    var e = document.createElement(name);
    if(css) e.setAttribute("class", css);
    return e;
};

var div = function(css)
{
    return elem("div", css);
};

var txt = function(t)
{
    return document.createTextNode(t);
};

var infohash_to_id = function(infohash)
{
    return "torrent_" + infohash;
};

function UI(xd)
{
    this._xd = xd;
}

/** build ui markup tree */
UI.prototype.build = function(root)
{
    var self = this;
    self.elems = {
        root: root,
        nav: self.buildNavbar(),
        torrents: self.buildTorrentsContainer()
    };
    console.log(self.elems);
    self.elems.root.appendChild(self.elems.nav);
    self.elems.root.appendChild(self.elems.torrents);
};

UI.prototype.buildTorrentRow = function(t)
{
    console.log("build torrent row for "+t.Infohash);
    var root = div("row");
    var id = infohash_to_id(t.Infohash);
    root.setAttribute("id", id);
    var widget = div("col-md-2");
    widget.setAttribute("id", id+"_widget");
    widget.appendChild(txt(util.bitfield_percent(t.Bitfield)));
    var nameText = txt(t.Name);
    var name = div("col-md-8");
    name.appendChild(nameText);
    var extra = div("col-md-2");
    root.appendChild(widget);
    root.appendChild(name);
    root.appendChild(extra);
    return root;
};

UI.prototype.buildNavbar = function()
{
    var self = this;
    var url_id = "xd-url-input";
    var nav = elem("nav", "navbar navbar-expand-md navbar-dark fixed-top");
    var inner = div("col-md-8 col-md-offset-2");
    nav.appendChild(inner);
    var label = elem("label");
    label.appendChild(txt("Url:"));
    label.setAttribute("for", url_id);
    inner.appendChild(label);
    var input = elem("input");
    input.setAttribute("id", url_id);
    inner.appendChild(input);
    var button = elem("button", "navbar-button btn btn-primary");
    button.appendChild(txt("Add Torrent"));
    inner.appendChild(button);
    button.onclick = function() {
        button.innerHTML = "Downloading...";
        var url = input.value;
        console.log("add torrent by url: "+url);
        self._xd.addTorrentByURL(url, function(err) {
            if(err) button.innerHTML = err;
            else {
                button.innerHTML = "Add Torrent";
                input.value = "";
            }
        });
    };
    return nav;
};

UI.prototype.buildTorrentsContainer = function()
{
    return div("container");
};

UI.prototype.hasTorrent = function(infohash)
{
    return document.getElementById(infohash_to_id(infohash)) != null;
};

UI.prototype.updateTorrent = function(t)
{
    // this should not fail
    var e = document.getElementById(infohash_to_id(t.Infohash) + "_widget");
    e.innerHTML = util.bitfield_percent(t.Bitfield) + "% " + util.peers_speed_string(t.Peers);
};

/**
   make sure a torrent in the ui exists given its info object, will not create a new cell if it's already there
   will update existing cell if it's there
 */
UI.prototype.ensureTorrentWithInfo = function(info)
{
    console.log(info);
    var self = this;
    if(!self.hasTorrent(info.Infohash)) {
        var e = self.buildTorrentRow(info);
        self.elems.torrents.appendChild(e);
    }
    self.updateTorrent(info);
};

/** lock ui and prepare for info update */
UI.prototype.beginTorrentUpdate = function()
{
    var self = this;
};

/** roll back and ui changes during an info update */
UI.prototype.rollbackTorrentUpdate = function()
{
    var self = this;
};

/** commit info update */
UI.prototype.commitTorrentUpdate = function()
{
    var self = this;
};

module.exports = {
    "UI": UI
};
