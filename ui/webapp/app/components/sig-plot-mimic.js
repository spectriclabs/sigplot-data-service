import Component from '@glimmer/component';
import { action } from '@ember/object';

import { Plot } from 'sigplot';

export default class SigPlotMimicComponent extends Component {
    
    constructor(owner, args) {
        super(owner, args);
        this._mimic = args.mimic;
        this._sources = [];
        this._listeners = [];

        // if you set no mask args, the default will be zoom=true, unzoom=true, pan=true
        this.mask = {}

        this.mask.xzoom = args.xzoom;
        this.mask.yzoom = args.yzoom;
        this.mask.zoom = args.zoom;
        this.mask.unzoom = args.unzoom;
        this.mask.pan = args.pan;
        this.mask.xpan = args.xpan;
        this.mask.ypan = args.ypan;

        // if neither xzoom or yzoom are set then use zoom parameters
        // if you set zoom and xzoom you get goth
        if (!this.mask.xzoom && !this.mask.yzoom) {
            this.mask.zoom = (args.zoom !== undefined) ? args.zoom : true;
            this.mask.unzoom = (args.unzoom !== undefined) ? args.unzoom : true;
        }

        // same for pan
        if (!this.mask.xpan && !this.mask.ypan) {
            this.mask.pan = (args.pan !== undefined) ? args.pan : true;
        }
    }

    @action
    source(plotcomp) {
        if (!this._sources.includes(plotcomp)) {
            this._sources.push(plotcomp);
        }
        for (let p of this._listeners) {
            // don't mimic ourselves
            if (p != plotcomp) {
                p.plot.mimic(plotcomp.plot, this.mask);
            }
        }
    }

    @action
    listener(plotcomp) {
        if (!this._listeners.includes(plotcomp)) {
            this._listeners.push(plotcomp);
            for (let p of this._sources) {
                // don't mimic ourselves
                if (p != plotcomp) {
                    plotcomp.plot.mimic(p.plot, this.mask);
                }
            }
        }
    }

    @action
    bidir(plotcomp) {
        this.listener(plotcomp);
        this.source(plotcomp);
    }

}
