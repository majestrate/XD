function bytesToSize(bytes) {
   var sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
   if (bytes == 0) return '0 B';
   var i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
   return Math.round(bytes / Math.pow(1024, i), 2) + ' ' + sizes[i];
}

var Torrent = function(data) {
    this.Name = data.Name;
    this.State = data.State;
    this.Infohash = data.Infohash;
    this.Peers = function() { return data.Peers ? data.Peers.length : 0; };
    this.Speed = function() {
        var tx = 0, rx = 0;
        if (data.Peers)
            data.Peers.forEach(function(p){tx += p.TX; rx += p.RX;});
        return "↑ " + bytesToSize(tx) +"/s ↓ " + bytesToSize(rx) + "/s";
    };
    this.RX = function() {
      var rx = 0;
      if(data.Peers) data.Peers.forEach(function(p) { rx += p.RX; });
      return rx;
    }
    this.TX = function() {
      var tx = 0;
      if(data.Peers) data.Peers.forEach(function(p) { tx += p.TX; });
      return tx;
    }
    this.TotalSize = function() {
        var total_size = 0;
        data.Files.forEach(function(f){ total_size += f.FileInfo.Length });
        return bytesToSize(total_size);
    };
    this.Progress = data.Progress * 100;
    this.remove = function() {
        viewModel._apicall({method: "XD.ChangeTorrent", action: "delete", infohash: this.Infohash, swarm: "0"}, function(data){
            console.log(data);
        });
    }.bind(this);
}
 
var viewModel = {
    _url: window.location.protocol + "//" + window.location.host + "/ecksdee/api",
    _apicall: function(call, cb)
    {
        $.ajax({
            type: "POST",
            url: this._url,
            contentType: "text/json; charset=UTF-8",
            data: JSON.stringify(call),
            success: function(j, text, xhr) {
                // console.log(call, j);
                cb(JSON.parse(j));
            }
        });
    },
    torrents: ko.observableArray(),
    torrentURL: ko.observable(),
    torrentFilter: ko.observable('all').bind(this),
    setFilter: function(state) {
        this.torrentFilter(state); main(); },
    addTorrent: function()
    {
        var _this = this;
        this._apicall({method: "XD.AddTorrent", swarm: "0", url: this.torrentURL()}, function(data){
            if (!data.error) _this.torrentURL("");
        });
    },
    torrentStates: ['all', 'downloading', 'seeding'],
    globalSpeed: function()
    {
      var rx = 0;
      var tx = 0;
      this.torrents().forEach(function(t) {
        rx += t.RX();
        tx += t.TX();
      });
      return  "Global: ↑ " + bytesToSize(tx) +"/s ↓ " + bytesToSize(rx) + "/s";
    }
};

function main()
{
    viewModel._apicall({method: "XD.SwarmStatus"}, function(data){
        viewModel.torrents.removeAll();
        for (var prop in data) {
            if (viewModel.torrentFilter() != 'all' & viewModel.torrentFilter() != data[prop].State)
                continue;
            viewModel.torrents.push(new Torrent(data[prop]));
        }
    });
}

window.onload = function()
{
    ko.applyBindings(viewModel);
    main(); setInterval(main, 5000);
}
