import Service from '@ember/service';
import fetch from 'fetch';
import config from '../config/environment';
import { inject as service } from '@ember/service';

export default class SdsService extends Service {
    @service notify;

    url = config.APP.SDS_URL + "/sdsdata/" // TODO we shouldn't hardcode the /sdsdata/ URL

    async getFiles() {
        try {
            const response = await fetch(this.url);
            if (response.ok) {
                const files = await response.json();
                return files;
            } else {
                return { files: [] };
            }
        } catch (e) {
            this.notify.error("failed to fetch from SDS server")
            return { files: [] };
        }
    }

    getFileUrl(file) {
        return this.url + file;
    }
}
