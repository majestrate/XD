/** Client-side UI Model */

var $ = require("jquery");

/**
 @param root element to inject CUM into
 */
function CUM(root, prepend)
{
    this._prepend = prepend || false;
    this._root = root;
    $(root).hide();
    this._elems = {};
}


// PUBLIC API

/** @param inject ui from json array */
CUM.prototype.inject = function(ja)
{
    var self = this;
    $(ja).each(function(idx, j) {
        try {
            var parent = null;
            if(j.parentID) {
                parent = $("#"+j.parentID) || null;
            } else if(j.parent) {
                parent = j.parent;
            }
            var e = self._newElem(j.name, j.css, j.id, j.attrs, parent);
            if(j.click) {
                e.onclick = function(ev) { j.click(ev); };
            }
            if(j.html) {
                $(e).append($(j.html));
            } else if(j.text) {
                $(e).text(j.text);
            } else

            if(j.finish && j.id) {
                self._elems[j.id] = function() { j.finish(e); };
            }
        } catch(ex) {
            console.log(idx, j, ex);
        }
    });
    return self;
};

/** @param finish building UI */
CUM.prototype.finish = function()
{
    var self = this;
    for (var id in self._elems) {
        var f = self._elems[id];
        if(f) f();
    }
    $(self._root).show();
};

// INTERNAL API

/**
 create new element <name class="css"></name>
 append to parent if provided otherwise append to root element
 @return created element
 */
CUM.prototype._newElem = function(name, css, id, attrs, parent)
{
    var self = this;
    var elem = document.createElement(name);
    if(css) {
        $(elem).addClass(css);
    }
    if(id) {
        $(elem).attr("id", id);
    }

    if(attrs) {
        for (var k in attrs) {
            $(elem).attr(k, attrs[k]);
        }
    }

    if (!parent) {
        parent = self._root;
    }
    if(self._prepend) {
        $(parent).prepend(elem);
    } else {
        $(parent).append(elem);
    }
    return elem;
};

module.exports = {
    "CUM" : CUM,
};
