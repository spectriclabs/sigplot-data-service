import Controller from '@ember/controller';
import { action } from '@ember/object';
import { tracked } from '@glimmer/tracking';
import { inject as service } from '@ember/service';

export default class DemoController extends Controller {
    @service notify;
    @service sds;

    @tracked sdsHref = null;
    @tracked rawHref = null;
    @tracked location = null;
    @tracked files = null;

    init() {
        super.init(...arguments);
    }

    @action
    plotFile(file) {
        this.sdsHref = this.sds.getFileUrl(file,"hdr",this.location);
        this.rawHref = this.sds.getFileUrl(file, "fs",this.location);
    }
    @action
    async setLocation(location) {
        this.location = location;
        this.files = await this.sds.getFiles(this.location);
    }

}
