'use strict';

// TODO how can we figure out to get this to run within the SDS GO APP behind a reverse proxy?
let SDS_URL = "/sigplot/sds";

if (process.env.SDS_URL) {
  SDS_URL = process.env.SDS_URL;
}

let ROOT_URL = "/sigplot/ui/";
if (process.env.ROOT_URL) {
  ROOT_URL = process.env.ROOT_URL;
}

module.exports = function(environment) {
  let ENV = {
    modulePrefix: 'sds-ui',
    environment,
    rootURL: ROOT_URL,
    locationType: 'auto',
    EmberENV: {
      FEATURES: {
        // Here you can enable experimental features on an ember canary build
        // e.g. EMBER_NATIVE_DECORATOR_SUPPORT: true
      },
      EXTEND_PROTOTYPES: {
        // Prevent Ember Data from overriding Date.parse.
        Date: false
      }
    },

    APP: {
      // Here you can pass flags/options to your application instance
      // when it is created
      SDS_URL: SDS_URL
    }
  };

  if (environment === 'development') {
    // ENV.APP.LOG_RESOLVER = true;
    // ENV.APP.LOG_ACTIVE_GENERATION = true;
    // ENV.APP.LOG_TRANSITIONS = true;
    // ENV.APP.LOG_TRANSITIONS_INTERNAL = true;
    // ENV.APP.LOG_VIEW_LOOKUPS = true;
  }

  if (environment === 'test') {
    // Testem prefers this...
    ENV.locationType = 'none';

    // keep test console output quieter
    ENV.APP.LOG_ACTIVE_GENERATION = false;
    ENV.APP.LOG_VIEW_LOOKUPS = false;

    ENV.APP.rootElement = '#ember-testing';
    ENV.APP.autoboot = false;
  }

  if (environment === 'production') {
    // here you can enable a production-specific feature
  }

  return ENV;
};
