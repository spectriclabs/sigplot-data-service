import React, { Suspense, lazy, useEffect } from 'react';

// @pierre/trees is the largest dependency in the app; load it in its own
// chunk so the shell paints without waiting for it.
const loadDataTree = () => import('./DataTree');
const DataTree = lazy(loadDataTree);

export default function FileBrowser({ locations, locationsStatus, onRetryLocations, onSelectFile }) {
  const locationList = Array.isArray(locations) ? locations : [];

  useEffect(() => {
    loadDataTree().catch(() => {});
  }, []);

  let body;
  if (locationsStatus === 'error') {
    body = (
      <div className="notice notice-error" role="alert">
        <span>Failed to load locations.</span>
        <button type="button" className="text-btn" onClick={onRetryLocations}>
          Retry
        </button>
      </div>
    );
  } else if (locationList.length > 0) {
    body = (
      <Suspense fallback={<p className="notice">Loading files...</p>}>
        <DataTree locations={locationList} onSelectFile={onSelectFile} />
      </Suspense>
    );
  } else if (locationsStatus === 'success') {
    body = <p className="notice">No locations configured</p>;
  } else {
    body = <p className="notice">Loading locations...</p>;
  }

  return (
    <div className="browser">
      <h2 className="eyebrow browser-heading">Data</h2>
      {body}
    </div>
  );
}
