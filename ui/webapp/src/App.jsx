import React, { useState, useEffect, useCallback, lazy, Suspense } from 'react';
import FileBrowser from './components/FileBrowser';
import { getLocations, getFiles, getFileUrl } from './api/sds';

const SigPlotViewer = lazy(() => import('./components/SigPlotViewer'));

export default function App() {
  const [locations, setLocations] = useState([]);
  const [locationsStatus, setLocationsStatus] = useState('loading');
  const [selectedLocation, setSelectedLocation] = useState('');
  const [files, setFiles] = useState([]);
  const [filesStatus, setFilesStatus] = useState('idle');
  const [path, setPath] = useState('');
  const [rawHref, setRawHref] = useState(null);
  const [sdsHref, setSdsHref] = useState(null);

  const fetchLocations = useCallback(async (signal) => {
    setLocationsStatus('loading');
    try {
      const locs = await getLocations({ signal });
      setLocations(locs);
      setLocationsStatus('success');
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

  const handleSelectLocation = useCallback((loc) => {
    setSelectedLocation(loc);
    setPath('');
    setRawHref(null);
    setSdsHref(null);
  }, []);

  const handleSelectFile = useCallback(
    (file) => {
      if (file.type === 'directory') {
        setPath((prev) => (prev ? `${prev}/${file.filename}` : file.filename));
      } else {
        const filePath = path ? `${path}/${file.filename}` : file.filename;
        setRawHref(getFileUrl(filePath, 'fs', selectedLocation));
        setSdsHref(getFileUrl(filePath, 'hdr', selectedLocation));
      }
    },
    [selectedLocation, path]
  );

  const handleGoBack = useCallback(() => {
    setPath((prev) => prev.split('/').slice(0, -1).join('/'));
  }, []);

  return (
    <div className="app">
      <header>
        <h1>SigPlot Data Service</h1>
      </header>
      <main>
        <FileBrowser
          locations={locations}
          locationsStatus={locationsStatus}
          selectedLocation={selectedLocation}
          onSelectLocation={handleSelectLocation}
          onRetryLocations={() => fetchLocations()}
          files={files}
          filesStatus={filesStatus}
          onSelectFile={handleSelectFile}
          path={path}
          onGoBack={handleGoBack}
        />
        <Suspense fallback={<div className="plot-container">Loading plots...</div>}>
          <SigPlotViewer rawHref={rawHref} sdsHref={sdsHref} />
        </Suspense>
      </main>
    </div>
  );
}
