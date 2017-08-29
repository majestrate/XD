/** ui.js -- ui builder */
var cum = require("./cum.js").CUM;
var $ = require("jquery");

function UI()
{
    // torrent cells in draw order
    this._torrentInfos = [];
}

/** build ui markup tree */
UI.prototype.build = function(elem)
{

};

/**
   make sure a torrent in the ui exists given its info object, will not create a new cell if it's already there
   will update existing cell if it's there
 */
UI.prototype.ensureTorrentWithInfo = function(info)
{
    var self = this;
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
