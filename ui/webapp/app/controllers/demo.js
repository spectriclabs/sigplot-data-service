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
    @tracked path = ""
    @tracked lastpath = [""]
    @tracked currentDepth = 0

    init() {
        super.init(...arguments);
    }

    @action
    plotFile(file) {
        if (file.type  == "file") {
            this.sdsHref = this.sds.getFileUrl(file.filename,"hdr",this.location+this.path);
            this.rawHref = this.sds.getFileUrl(file.filename, "fs",this.location+this.path);
        } else if (file.type =="directory") {
            this.lastpath[this.currentDepth+1] = this.path;
            this.currentDepth++;
            this.path = this.path+"/" + file.filename; 
            this.setLocation(this.location);
        }


    }
    @action
    async setLocation(location) {
        this.location = location;
        this.files = await this.sds.getFiles(this.location+this.path);
    }

    @action
    goback() {
        if (this.currentDepth !=0) {
            this.path=this.lastpath[this.currentDepth];
            this.currentDepth--;
            this.setLocation(this.location);
        }

    }
}
