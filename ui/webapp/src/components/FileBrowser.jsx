import React, { useEffect, useMemo, useState } from 'react';
import { isBlueFile } from '../api/sds';
import {
  ArrowUpIcon,
  ChevronRightIcon,
  FileIcon,
  FolderIcon,
  SearchIcon,
} from './icons';

function joinPath(dir, name) {
  return dir ? `${dir}/${name}` : name;
}

const byTypeThenName = (a, b) => {
  const aDir = a.type === 'directory';
  const bDir = b.type === 'directory';
  if (aDir !== bDir) return aDir ? -1 : 1;
  return a.filename.localeCompare(b.filename, undefined, { numeric: true, sensitivity: 'base' });
};

function LocationPicker({ locations, status, selected, onSelect, onRetry }) {
  if (status === 'error') {
    return (
      <div className="notice notice-error" role="alert">
        <span>Failed to load locations.</span>
        <button type="button" className="text-btn" onClick={onRetry}>
          Retry
        </button>
      </div>
    );
  }
  if (status === 'success' && locations.length === 0) {
    return <p className="notice">No locations found</p>;
  }
  const loading = status === 'loading' && locations.length === 0;
  return (
    <div className="select-wrap">
      <select
        id="location-select"
        className="select"
        value={selected}
        disabled={loading}
        onChange={(e) => onSelect(e.target.value)}
      >
        {loading && <option value="">Loading locations...</option>}
        {!loading && !selected && (
          <option value="" disabled>
            Choose a location
          </option>
        )}
        {locations.map((loc) => (
          <option key={loc} value={loc}>
            {loc}
          </option>
        ))}
      </select>
    </div>
  );
}

function FileRow({ file, selected, onSelect }) {
  const isDir = file.type === 'directory';
  return (
    <li>
      <button
        type="button"
        className={`file-row${selected ? ' is-selected' : ''}${isDir ? ' is-dir' : ''}`}
        aria-current={selected ? 'true' : undefined}
        title={file.filename}
        onClick={() => onSelect(file)}
      >
        {isDir ? <FolderIcon className="file-row-icon" /> : <FileIcon className="file-row-icon" />}
        <span className="file-row-name">{file.filename}</span>
        {isDir && <ChevronRightIcon className="file-row-chevron" size={14} />}
      </button>
    </li>
  );
}

export default function FileBrowser({
  locations,
  locationsStatus,
  selectedLocation,
  onSelectLocation,
  onRetryLocations,
  files,
  filesStatus,
  onSelectFile,
  selectedFile,
  path,
  onGoBack,
  onNavigate,
}) {
  const locationList = Array.isArray(locations) ? locations : [];
  const segments = path ? path.split('/') : [];
  const [filter, setFilter] = useState('');

  useEffect(() => {
    setFilter('');
  }, [selectedLocation, path]);

  // Only folders and BLUE files can be browsed or plotted; anything else
  // (source, config) would just fail to load.
  const sortedFiles = useMemo(
    () =>
      Array.isArray(files)
        ? files.filter((f) => f.type === 'directory' || isBlueFile(f.filename)).sort(byTypeThenName)
        : [],
    [files]
  );
  const query = filter.trim().toLowerCase();
  const visibleFiles = query
    ? sortedFiles.filter((f) => f.filename.toLowerCase().includes(query))
    : sortedFiles;
  const ready = filesStatus === 'success';

  return (
    <div className="browser">
      <div className="browser-section">
        <label className="eyebrow" htmlFor="location-select">
          Location
        </label>
        <LocationPicker
          locations={locationList}
          status={locationsStatus}
          selected={selectedLocation}
          onSelect={onSelectLocation}
          onRetry={onRetryLocations}
        />
      </div>

      {selectedLocation && (
        <div className="browser-files">
          <div className="pathbar">
            <button
              type="button"
              className="icon-btn"
              aria-label="Up one folder"
              title="Up one folder"
              disabled={!path}
              onClick={onGoBack}
            >
              <ArrowUpIcon />
            </button>
            <nav className="crumbs" aria-label="Current folder">
              <ol>
                <li>
                  <button
                    type="button"
                    aria-current={segments.length === 0 ? 'location' : undefined}
                    onClick={() => onNavigate('')}
                  >
                    {selectedLocation}
                  </button>
                </li>
                {segments.map((segment, i) => {
                  const target = segments.slice(0, i + 1).join('/');
                  return (
                    <li key={target}>
                      <button
                        type="button"
                        aria-current={i === segments.length - 1 ? 'location' : undefined}
                        onClick={() => onNavigate(target)}
                      >
                        {segment}
                      </button>
                    </li>
                  );
                })}
              </ol>
            </nav>
          </div>

          <div className="search">
            <SearchIcon className="search-icon" size={14} />
            <input
              type="search"
              className="search-input"
              placeholder="Filter files"
              aria-label="Filter files"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              disabled={!ready || sortedFiles.length === 0}
            />
          </div>

          <div className="file-scroll">
            {filesStatus === 'loading' && <p className="notice">Loading files...</p>}
            {filesStatus === 'error' && (
              <p className="notice notice-error" role="alert">
                Failed to load files.
              </p>
            )}
            {ready && sortedFiles.length === 0 && <p className="notice">No files found</p>}
            {ready && sortedFiles.length > 0 && visibleFiles.length === 0 && (
              <p className="notice">No files match “{filter.trim()}”</p>
            )}
            {ready && visibleFiles.length > 0 && (
              <ul className="file-list" aria-label="Files">
                {visibleFiles.map((file) => (
                  <FileRow
                    key={file.filename}
                    file={file}
                    selected={
                      file.type !== 'directory' &&
                      joinPath(path, file.filename) === selectedFile
                    }
                    onSelect={onSelectFile}
                  />
                ))}
              </ul>
            )}
          </div>

          {ready && sortedFiles.length > 0 && (
            <p className="browser-footer" aria-live="polite">
              {query
                ? `${visibleFiles.length} of ${sortedFiles.length} items`
                : `${sortedFiles.length} ${sortedFiles.length === 1 ? 'item' : 'items'}`}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
