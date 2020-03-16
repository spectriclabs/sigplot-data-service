import Service from '@ember/service';
import fetch from 'fetch';
import config from '../config/environment';

export default class SdsService extends Service {
    url = config.APP.SDS_URL + "/sdsdata/" // TODO we shouldn't hardcode the /sdsdata/ URL

    async getFiles() {
        const response = await fetch(this.url);
        if (response.ok) {
            const files = await response.json();
            return files;
        } else {
            return { files: [] };
        }
    }

    getFileUrl(file) {
        return this.url + file;
    }
}
