import Service from '@ember/service';
import fetch from 'fetch';
import config from '../config/environment';
import { inject as service } from '@ember/service';

export default class SdsService extends Service {
    @service notify;

    url = config.APP.SDS_URL  // TODO we shouldn't hardcode the /sdsdata/ URL

    async getFiles(location) {
        try {
            const response = await fetch(this.url+ "/fs/"+location+"/");
            if (response.ok) {
                const fileInfo = await response.json();
                return fileInfo
            } else {
                return { files: [] };
            }
        } catch (e) {
            this.notify.error("failed to fetch from SDS server")
            return { files: [] };
        }
    }

    async getLocations() {
        try {
            const response = await fetch(this.url+ "/fs/");
            if (response.ok) {
                const locationInfo = await response.json();
                var locations = []
                for (var locationNum=0;locationNum<locationInfo.length;locationNum++) {
                    locations.push(locationInfo[locationNum].locationName)
                }
                return locations;
            } else {
                return { locations: [] };
            }
        } catch (e) {
            this.notify.error("failed to fetch from SDS server")
            return { locations: [] };
        }
    }

    getFileUrl(file,mode,location) {
        return this.url +"/"+ mode + "/" + location + "/" + file;
    }
}
