import Route from '@ember/routing/route';
import { task, timeout } from 'ember-concurrency';
import { inject as service } from '@ember/service';

export default class DemoRoute extends Route {
    @service sds;
    @service notify;

    async model() {
        const files = await this.sds.getFiles();
        //this.notify.info(`loaded ${files.length} files`);
        return { files  };
    }

    setupController() {
        super.setupController(...arguments);
        this.get("pollServerForChanges").perform();
    }

    deactivate() {
        this.get("pollServerForChanges").cancel();
    }

    @(task(function * () {
        yield timeout(500);
        try {
          while (true) {
            // refresh the model every 5 seconds
            yield timeout(5000);
            this.refresh();
          }
        } finally {
            // pass
        }
      })).restartable() pollServerForChanges;
    
}
