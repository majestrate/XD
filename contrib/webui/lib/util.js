
// utils for bitfields

var _bitfield_has = function(bfdata, idx)
{
    return bfdata[idx >> 3] & (1 << ((7 - idx) & 7)) != 0;
};

var bitfield_count_set = function(bf)
{
    var set = 0;
    for( var idx = 0; idx < bf.length; idx++) {
        if(bf[idx])
            set ++;
    }
    console.log(set);
    return set;
};

var bitfield_to_percent = function(bf)
{
    return parseInt(""+((bitfield_count_set(bf) / bf.length) * 10000)) / 100;
};

var unitify = function(n)
{
    if (n > 1024 * 1024) {
        n /= 1024 * 1024;
        n = parseInt(n * 100) / 100;
        n += "M";
        return n;
    } else if (n > 1024) {
        n /= 1024;
        n = parseInt(n * 100) / 100;
        n += "K";
        return n;
    } else {
        return n;
    }
};

var peers_speed = function(peers)
{
    if(!peers) return "0 / 0";
    var tx = 0;
    var rx = 0;
    for(var idx = 0; idx < peers.length ; idx++)
    {
        tx += peers[idx].TX;
        rx += peers[idx].RX;
    }

    return "tx " + unitify(tx ) + "B/s rx " + unitify(rx) + "B/s";
};

module.exports = {
    "bitfield_percent": bitfield_to_percent,
    "bitfield_count_set": bitfield_count_set,
    "unitify": unitify,
    "peers_speed_string": peers_speed
};
