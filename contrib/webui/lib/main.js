function bytesToSize(bytes) {
   var sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
   if (bytes == 0) return '0 B';
   var i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
   return Math.round(bytes / Math.pow(1024, i), 2) + ' ' + sizes[i];
}

function formatFloat(f, eps) {
  if (!eps) eps = 2;
  eps = Math.pow(10, eps);
  return parseInt(f * eps) / eps;
}

function makeRatio(tx, rx) {
  var r = "0.0";
  if ( rx > 0 ) {
    if ( tx > 0 ) {
      r = "" + formatFloat(tx / rx);
    }
	} else if ( tx > 0 ) {
		r = "\u221E";
	}
  return r;
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
        return "\u2191 " + bytesToSize(tx) +"/s \u2193 " + bytesToSize(rx) + "/s";
    };
    this.RX = function() {
      var rx = 0;
      if(data.Peers) data.Peers.forEach(function(p) { rx += p.RX; });
      return rx;
    };
    this.TX = function() {
      var tx = 0;
      if(data.Peers) data.Peers.forEach(function(p) { tx += p.TX; });
      return tx;
    };
    this.Data = function() {
      return data;
    };
    this.TotalSize = function() {
      var total_size = 0;
      if(data.Files) {
        data.Files.forEach(function(f){ total_size += f.FileInfo.Length });
      }
      return bytesToSize(total_size);
    };
    this.Progress = formatFloat(data.Progress * 100);
    this.Ratio = function() {
      return "("+makeRatio(data.TX, data.RX) + " ratio)";
    };

  this.changeTorrent = function(action)
  {
    viewModel._apicall({method: "XD.ChangeTorrent", action: action, infohash: this.Infohash, swarm: "0"}, function(data){
      console.log(data);
    });    
  }.bind(this);

  this.remove = function()
  {
    if (viewModel.confirmation.silent()) {
        viewModel.deleteTorrent(this.Infohash, viewModel.confirmation.deleteFiles());
    } else {
        viewModel.confirmation.Infohash(this.Infohash);
        viewModel.confirmation.show(true);
    }
  }.bind(this);

  this.start = function()
  {
    this.changeTorrent("start");
  }.bind(this);

  this.stop = function()
  {
    this.changeTorrent("stop");
  }.bind(this);

  this.toggle = function()
  {
    if(this.Stopped())
      this.start();
    else
      this.stop();
  }.bind(this);

  this.Stopped = function()
  {
    return data.State == "stopped";
  };
  
  this.StatusButton = function()
  {
    if (this.Stopped())
    {
      return "\u25BA";
    }
    else
    {
      return "\u275A\u275A";
    }
  };
}

var viewModel = {
    _url: "ecksdee/api",
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
    deleteTorrent: function(Infohash, deleteFiles)
    {
        var action = deleteFiles ? "delete" : "remove";
        this._apicall({
                method: "XD.ChangeTorrent",
                action: action,
                infohash: Infohash, swarm: "0"},
            function(data){ console.log(data); });
    },
    torrentStates: ['all', 'downloading', 'seeding'],
    globalInfo: function()
    {
        var rx = 0;
        var tx = 0;
        var peers = 0;
        var rtx = 0;
        var rrx = 0;
        var count = 0;
        this.torrents().forEach(function(t) {
            rx += t.RX();
            tx += t.TX();
            rrx += t.Data().RX;
            rtx += t.Data().TX;
            peers += t.Peers();
            count ++;
        });
        return peers+" peers connected on "+ count+ " torrents (" + makeRatio(rtx, rrx) + " ratio) \u2191 " + bytesToSize(tx) +"/s \u2193 " + bytesToSize(rx) + "/s";
    },

    // confirmation box
    confirmation: {
        Infohash: ko.observable(),
        show: ko.observable(false), silent: ko.observable(false), deleteFiles: ko.observable(false),
        close: function() { this.confirmation.Infohash(null);
            this.confirmation.silent(false); this.confirmation.show(false); },
        confirmed: function() {
            this.deleteTorrent(this.confirmation.Infohash(), this.confirmation.deleteFiles());
            this.confirmation.Infohash(null); this.confirmation.show(false); }
    },
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
    main(); setInterval(main, 1000);
}
