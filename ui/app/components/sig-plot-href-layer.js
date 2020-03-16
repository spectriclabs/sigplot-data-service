import Component from '@glimmer/component';
import { tracked } from '@glimmer/tracking';
import { action } from '@ember/object';

export default class SigPlotHrefLayerComponent extends Component {
    @tracked lyrN = null;
    
    constructor(owner, args) {
        super(owner, args);
        this._plot = args.plot;
        this.lyrN = null;
    }

    @action
    onInsert() {
        this.overlay(this.args.href);
    }
    
    @action
    reload() {
        this.overlay(this.args.href);
    }

    overlay(href) {
        // if the plot doesn't exist, nothing to do
        if (this._plot === null) {
            return;
        }

        // deoverlay first
        if (this.lyrN !== null) {
            this._plot.deoverlay(this.lyrN);
            this.lyrN = null;
        }

        // if href isnt set we are done
        if (!href) {
            return
        }

        // add the overlay
        let options = {};
        if (this.args.layerType) {
            options.layerType = this.args.layerType;
        }
        if (this.args.href) {
            this.lynN = this._plot.plot.overlay_href(this.args.href, null, options);
        }
    }

}
