import React, { useRef } from 'react';
import { FileTree, useFileTree, useFileTreeSearch } from '@pierre/trees/react';
import { getFiles } from '../api/sds';
import { splitTreePath, useLocationTree } from '../hooks/useLocationTree';

const TREE_HOST_STYLE = { height: '100%' };

export default function DataTree({ locations, onSelectFile }) {
  const latest = useRef({ locations, onSelectFile });
  latest.current = { locations, onSelectFile };

  const { model } = useFileTree({
    paths: [],
    search: true,
    fileTreeSearchMode: 'hide-non-matches',
    density: 'compact',
    icons: 'standard',
    onSelectionChange: (paths) => {
      const file = [...paths].reverse().find((path) => !path.endsWith('/'));
      const target = file && splitTreePath(file, latest.current.locations);
      if (target) latest.current.onSelectFile(target);
    },
  });
  const { folderError } = useLocationTree(model, locations, getFiles);
  const search = useFileTreeSearch(model);
  const query = search.value ?? '';
  const matches = search.matchingPaths.length;

  return (
    <div className="browser-files">
      {/* React 18 passes props to custom elements as raw attributes, so the
          class lives on a wrapper rather than the tree host. */}
      <div className="tree">
        <FileTree model={model} aria-label="Locations and files" style={TREE_HOST_STYLE} />
      </div>

      {folderError && (
        <p className="notice notice-error browser-footer" role="alert">
          Couldn’t open {folderError.replace(/\/$/, '')}. Expand it to retry.
        </p>
      )}
      {!folderError && query && (
        <p className="browser-footer" aria-live="polite">
          {matches} {matches === 1 ? 'match' : 'matches'} in loaded folders
        </p>
      )}
    </div>
  );
}
