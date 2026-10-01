import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import DataTree from '../components/DataTree';
import { getFiles } from '../api/sds';

vi.mock('../api/sds', async (importOriginal) => ({
  ...(await importOriginal()),
  getFiles: vi.fn(),
}));

const shadowRoot = (container) => container.querySelector('file-tree-container').shadowRoot;
const row = (container, path) =>
  shadowRoot(container).querySelector(`[data-item-path="${CSS.escape(path)}"]`);
const rowPaths = (container) =>
  [...shadowRoot(container).querySelectorAll('[role=treeitem]')].map((el) =>
    el.getAttribute('data-item-path')
  );

describe('DataTree', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders locations as top-level folders with the first one open', async () => {
    getFiles.mockResolvedValue([
      { filename: 'sub', type: 'directory' },
      { filename: 'a.tmp', type: 'file' },
      { filename: 'readme.md', type: 'file' },
    ]);
    const { container } = render(<DataTree locations={['A', 'B']} onSelectFile={vi.fn()} />);

    await waitFor(() => expect(rowPaths(container)).toEqual(['A/', 'A/sub/', 'A/a.tmp', 'B/']));
    expect(row(container, 'A/')).toHaveAttribute('aria-expanded', 'true');
    expect(row(container, 'B/')).toHaveAttribute('aria-expanded', 'false');
  });

  it('reports the selected file as a location and path', async () => {
    getFiles.mockResolvedValue([{ filename: 'a.tmp', type: 'file' }]);
    const onSelectFile = vi.fn();
    const { container } = render(<DataTree locations={['Loc']} onSelectFile={onSelectFile} />);
    await waitFor(() => expect(row(container, 'Loc/a.tmp')).not.toBeNull());

    fireEvent.click(row(container, 'Loc/a.tmp'));

    expect(onSelectFile).toHaveBeenCalledWith({ location: 'Loc', path: 'a.tmp' });
    await waitFor(() => expect(row(container, 'Loc/a.tmp')).toHaveAttribute('aria-selected', 'true'));
  });

  it('does not treat folder clicks as file selections', async () => {
    getFiles.mockImplementation(async (_location, path) =>
      path === '' ? [{ filename: 'sub', type: 'directory' }] : [{ filename: 'x.prm', type: 'file' }]
    );
    const onSelectFile = vi.fn();
    const { container } = render(<DataTree locations={['Loc']} onSelectFile={onSelectFile} />);
    await waitFor(() => expect(row(container, 'Loc/sub/')).not.toBeNull());

    fireEvent.click(row(container, 'Loc/sub/'));

    await waitFor(() => expect(row(container, 'Loc/sub/x.prm')).not.toBeNull());
    expect(getFiles).toHaveBeenLastCalledWith('Loc', 'sub', expect.any(Object));
    expect(onSelectFile).not.toHaveBeenCalled();
  });

  it('explains when a folder fails to open', async () => {
    getFiles.mockRejectedValue(new Error('offline'));
    render(<DataTree locations={['Down']} onSelectFile={vi.fn()} />);

    expect(await screen.findByRole('alert')).toHaveTextContent('Couldn’t open Down. Expand it to retry.');
  });
});
