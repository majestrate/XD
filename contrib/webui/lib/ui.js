/** ui.js -- ui builder */
var cum = require("./cum.js").CUM;
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


var infohash_to_id = function(infohash)
{
    return "torrent_" + infohash;
};

function UI()
{
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
    root.setAttribute("id", infohash_to_id(t.Infohash));
    var widget = div("col-md-2");
    var nameText = document.createTextNode(t.Name);
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
    var url_id = "xd-url-input";
    var nav = elem("nav", "navbar navbar-expand-md navbar-dark fixed-top");
    var inner = div("col-md-8 col-md-offset-2");
    nav.appendChild(inner);
    var label = elem("label");
    label.appendChild(document.createTextNode("RPC Server:"));
    label.setAttribute("for", url_id);
    inner.appendChild(label);
    var input = elem("input");
    input.setAttribute("id", url_id);
    inner.appendChild(input);
    var button = elem("button", "navbar-button btn btn-primary");
    button.appendChild(document.createTextNode("connect"));
    inner.appendChild(button);
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
    var e = document.getElementById(t.Infohash);

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
