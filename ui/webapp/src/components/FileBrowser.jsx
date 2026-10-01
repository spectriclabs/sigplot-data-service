import React from 'react';

export default function FileBrowser({
  locations,
  locationsStatus,
  selectedLocation,
  onSelectLocation,
  onRetryLocations,
  files,
  filesStatus,
  onSelectFile,
  path,
  onGoBack,
}) {
  const locationList = Array.isArray(locations) ? locations : [];
  const fileList = Array.isArray(files) ? files : [];

  return (
    <>
      <div className="location-panel">
        <h4>Choose Location:</h4>
        <ul className="list-group">
          {locationList.map((loc) => (
            <button
              key={loc}
              className={`list-group-item${loc === selectedLocation ? ' active' : ''}`}
              onClick={() => onSelectLocation(loc)}
            >
              {loc}
            </button>
          ))}
          {locationsStatus === 'loading' && locationList.length === 0 && (
            <p className="empty-message">Loading locations...</p>
          )}
          {locationsStatus === 'error' && (
            <p className="empty-message">
              Failed to load locations.
              <button type="button" onClick={onRetryLocations}>
                Retry
              </button>
            </p>
          )}
          {locationsStatus === 'success' && locationList.length === 0 && (
            <p className="empty-message">No locations found</p>
          )}
        </ul>
      </div>

      <div className="file-browser">
        <div className="file-browser-header">
          <h4>Choose File:</h4>
          {path && (
            <button className="back-btn" onClick={onGoBack}>
              Back
            </button>
          )}
        </div>
        {selectedLocation ? (
          <ul className="list-group">
            {filesStatus === 'loading' && (
              <p className="empty-message">Loading files...</p>
            )}
            {filesStatus === 'error' && (
              <p className="empty-message">Failed to load files.</p>
            )}
            {filesStatus === 'success' && fileList.map((file) => (
              <button
                key={file.filename}
                className="list-group-item"
                onClick={() => onSelectFile(file)}
              >
                <p className={`file-name ${file.type}`}>
                  {file.filename}
                </p>
              </button>
            ))}
            {filesStatus === 'success' && fileList.length === 0 && (
              <p className="empty-message">No files found</p>
            )}
          </ul>
        ) : (
          <p className="empty-message">Select a location first</p>
        )}
      </div>
    </>
  );
}
