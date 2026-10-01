import { useEffect, useState } from 'react';
import { isBlueFile } from '../api/sds';

const stripSlash = (path) => (path.endsWith('/') ? path.slice(0, -1) : path);

// Trees identifies items by canonical path; directories end with "/".
export function toTreePath(directory, entry) {
  const base = directory ? `${stripSlash(directory)}/` : '';
  return entry.type === 'directory' ? `${base}${entry.filename}/` : `${base}${entry.filename}`;
}

/**
 * Splits a tree path ("Loc/sub/file.tmp") into its SDS location and the path
 * inside that location. Matching on known roots keeps this correct even if a
 * location name contains a slash.
 */
export function splitTreePath(treePath, locations) {
  for (const location of locations) {
    const root = `${location}/`;
    if (treePath === root || treePath.startsWith(root)) {
      return { location, path: stripSlash(treePath.slice(root.length)) };
    }
  }
  return null;
}

/**
 * Shows every SDS location as a top-level folder in a Trees model and fills
 * folders from the listing API one directory at a time, the first time each
 * one is expanded. Only folders and BLUE files are shown, since nothing else
 * can be plotted. The first location starts expanded. In-flight requests are
 * aborted when the locations change or the component unmounts, and results
 * that arrive afterwards are ignored.
 */
export function useLocationTree(model, locations, listDirectory) {
  const [folderError, setFolderError] = useState(null);

  useEffect(() => {
    const loaded = new Set();
    const pending = new Map();
    const unloadedDirs = new Set(locations.map((location) => `${location}/`));
    let disposed = false;

    const load = async (dirPath) => {
      if (loaded.has(dirPath) || pending.has(dirPath)) return;
      const target = splitTreePath(dirPath, locations);
      if (!target) return;
      const controller = new AbortController();
      pending.set(dirPath, controller);
      try {
        const entries = await listDirectory(target.location, target.path, {
          signal: controller.signal,
        });
        if (disposed) return;
        const operations = [];
        for (const entry of Array.isArray(entries) ? entries : []) {
          if (entry.type !== 'directory' && !isBlueFile(entry.filename)) continue;
          const path = toTreePath(dirPath, entry);
          if (model.getItem(path)) continue;
          operations.push({ type: 'add', path });
          if (path.endsWith('/')) unloadedDirs.add(path);
        }
        loaded.add(dirPath);
        unloadedDirs.delete(dirPath);
        if (operations.length) model.batch(operations);
        setFolderError((current) => (current === dirPath ? null : current));
      } catch (error) {
        if (disposed || error.name === 'AbortError') return;
        setFolderError(dirPath);
        // Collapse so expanding again retries the request.
        model.getItem(dirPath)?.collapse();
      } finally {
        if (pending.get(dirPath) === controller) pending.delete(dirPath);
      }
    };

    const loadExpanded = () => {
      for (const dir of unloadedDirs) {
        const item = model.getItem(dir);
        if (item?.isDirectory() && item.isExpanded()) load(dir);
      }
    };

    setFolderError(null);
    model.resetPaths([...unloadedDirs]);
    const unsubscribe = model.subscribe(loadExpanded);
    if (locations.length) model.getItem(`${locations[0]}/`)?.expand();
    loadExpanded();

    return () => {
      disposed = true;
      unsubscribe();
      pending.forEach((controller) => controller.abort());
      pending.clear();
    };
  }, [model, locations, listDirectory]);

  return { folderError };
}
