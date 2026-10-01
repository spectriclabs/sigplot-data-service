import React, { useState, useEffect, useCallback, useMemo, lazy, Suspense } from 'react';
import FileBrowser from './components/FileBrowser';
import { ColumnsIcon, RowsIcon, SidebarIcon, WaveformIcon } from './components/icons';
import { getLocations, getFiles, getFileUrl } from './api/sds';

const loadSigPlotViewer = () => import('./components/SigPlotViewer');
const SigPlotViewer = lazy(loadSigPlotViewer);

function usePersistentState(key, initial) {
  const [value, setValue] = useState(() => {
    try {
      const stored = window.localStorage.getItem(key);
      return stored == null ? initial : JSON.parse(stored);
    } catch {
      return initial;
    }
  });
  useEffect(() => {
    try {
      window.localStorage.setItem(key, JSON.stringify(value));
    } catch {
      // Storage can be unavailable (private mode, quota); the UI still works.
    }
  }, [key, value]);
  return [value, setValue];
}

const NARROW_QUERY = '(max-width: 760px)';

const LAYOUTS = [
  { id: 'stacked', label: 'Stacked', Icon: RowsIcon },
  { id: 'columns', label: 'Side by side', Icon: ColumnsIcon },
];

function EmptyState({ hasLocation }) {
  return (
    <div className="empty-state" role="status">
      <div className="empty-state-icon">
        <WaveformIcon size={28} />
      </div>
      <p className="empty-state-title">No file selected</p>
      <p className="empty-state-text">
        {hasLocation
          ? 'Select a file from the sidebar to plot it.'
          : 'Choose a location, then select a file to plot it.'}
      </p>
    </div>
  );
}

export default function App() {
  const [locations, setLocations] = useState([]);
  const [locationsStatus, setLocationsStatus] = useState('loading');
  const [selectedLocation, setSelectedLocation] = useState('');
  const [files, setFiles] = useState([]);
  const [filesStatus, setFilesStatus] = useState('idle');
  const [path, setPath] = useState('');
  const [selectedFile, setSelectedFile] = useState(null);
  const [sidebarOpen, setSidebarOpen] = usePersistentState('sds.sidebarOpen', true);
  const [layout, setLayout] = usePersistentState('sds.plotLayout', 'stacked');

  const fetchLocations = useCallback(async (signal) => {
    setLocationsStatus('loading');
    try {
      const locs = await getLocations({ signal });
      setLocations(locs);
      setLocationsStatus('success');
      setSelectedLocation((current) => current || locs[0] || '');
    } catch (error) {
      if (error.name === 'AbortError') return;
      setLocations([]);
      setLocationsStatus('error');
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    fetchLocations(controller.signal);
    return () => controller.abort();
  }, [fetchLocations]);

  useEffect(() => {
    if (!selectedLocation) {
      setFiles([]);
      setFilesStatus('idle');
      return;
    }
    let ignore = false;
    const controller = new AbortController();
    const fetchFiles = async () => {
      setFilesStatus('loading');
      try {
        const fileList = await getFiles(selectedLocation, path, {
          signal: controller.signal,
        });
        if (ignore) return;
        setFiles(Array.isArray(fileList) ? fileList : []);
        setFilesStatus('success');
      } catch (error) {
        if (ignore || error.name === 'AbortError') return;
        setFiles([]);
        setFilesStatus('error');
      }
    };
    fetchFiles();
    return () => {
      ignore = true;
      controller.abort();
    };
  }, [selectedLocation, path]);

  // Fetch the plotting chunk while the user browses, so it's ready by the
  // time a file is picked.
  useEffect(() => {
    if (selectedLocation) loadSigPlotViewer().catch(() => {});
  }, [selectedLocation]);

  const handleSelectLocation = useCallback((loc) => {
    setSelectedLocation(loc);
    setPath('');
    setSelectedFile(null);
  }, []);

  const handleSelectFile = useCallback(
    (file) => {
      if (file.type === 'directory') {
        setPath((prev) => (prev ? `${prev}/${file.filename}` : file.filename));
      } else {
        setSelectedFile(path ? `${path}/${file.filename}` : file.filename);
        // On narrow screens the sidebar overlays the plots; get it out of the way.
        if (window.matchMedia?.(NARROW_QUERY).matches) setSidebarOpen(false);
      }
    },
    [path, setSidebarOpen]
  );

  const hrefs = useMemo(
    () =>
      selectedFile
        ? {
            raw: getFileUrl(selectedFile, 'fs', selectedLocation),
            sds: getFileUrl(selectedFile, 'hdr', selectedLocation),
          }
        : null,
    [selectedFile, selectedLocation]
  );

  const handleGoBack = useCallback(() => {
    setPath((prev) => prev.split('/').slice(0, -1).join('/'));
  }, []);

  const toggleLabel = sidebarOpen ? 'Hide file browser' : 'Show file browser';

  return (
    <div className={`app${sidebarOpen ? '' : ' is-collapsed'}`}>
      <header className="topbar">
        <button
          type="button"
          className="icon-btn"
          aria-controls="sidebar"
          aria-expanded={sidebarOpen}
          aria-label={toggleLabel}
          title={toggleLabel}
          onClick={() => setSidebarOpen((open) => !open)}
        >
          <SidebarIcon size={18} />
        </button>
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">
            <WaveformIcon size={16} />
          </span>
          <h1>SigPlot Data Service</h1>
        </div>
        <div className="segmented" role="group" aria-label="Plot layout">
          {LAYOUTS.map(({ id, label, Icon }) => (
            <button
              key={id}
              type="button"
              aria-pressed={layout === id}
              aria-label={label}
              title={label}
              onClick={() => setLayout(id)}
            >
              <Icon size={15} />
              <span>{label}</span>
            </button>
          ))}
        </div>
      </header>

      <div className="workspace">
        <aside id="sidebar" className="sidebar" aria-label="File browser" hidden={!sidebarOpen}>
          <FileBrowser
            locations={locations}
            locationsStatus={locationsStatus}
            selectedLocation={selectedLocation}
            onSelectLocation={handleSelectLocation}
            onRetryLocations={() => fetchLocations()}
            files={files}
            filesStatus={filesStatus}
            onSelectFile={handleSelectFile}
            selectedFile={selectedFile}
            path={path}
            onGoBack={handleGoBack}
            onNavigate={setPath}
          />
        </aside>
        <main className={`plots plots-${layout === 'columns' ? 'columns' : 'stacked'}`}>
          {hrefs ? (
            <Suspense
              fallback={
                <div className="empty-state" role="status">
                  <span className="sds-spinner" aria-hidden="true" />
                  <p className="empty-state-text">Loading plots...</p>
                </div>
              }
            >
              <SigPlotViewer rawHref={hrefs.raw} sdsHref={hrefs.sds} fileName={selectedFile} />
            </Suspense>
          ) : (
            <EmptyState hasLocation={Boolean(selectedLocation)} />
          )}
        </main>
      </div>
    </div>
  );
}
