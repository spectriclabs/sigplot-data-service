import Component from '@glimmer/component';
import { action } from '@ember/object';

import { Plot } from 'sigplot';

export default class SigPlotComponent extends Component {

    @action
    constructPlot(element) {
        this.plot = new Plot(element, this.args.options);
        if (this.args.mimic) {
            this.args.mimic(this);
        }
    }

    @action
    destroyPlot() {
        // TODO anything necessary here?
    }

}
