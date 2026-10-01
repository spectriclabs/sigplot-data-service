import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen, waitFor, within } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import App from '../App';
import { getLocations, getFiles, getFileUrl } from '../api/sds';

vi.mock('../api/sds', async (importOriginal) => ({
  ...(await importOriginal()),
  getLocations: vi.fn(),
  getFiles: vi.fn(),
  getFileUrl: vi.fn((file, mode, location) => `/mock/${mode}/${location}/${file}`),
}));

vi.mock('../components/SigPlotViewer', () => ({
  default: ({ rawHref, sdsHref, fileName }) => (
    <div>
      <output data-testid="raw-href">{rawHref || ''}</output>
      <output data-testid="sds-href">{sdsHref || ''}</output>
      <output data-testid="file-name">{fileName || ''}</output>
    </div>
  ),
}));

// Node 25+ exposes its own (unconfigured) localStorage global that shadows
// jsdom's, so tests install a deterministic in-memory Storage instead.
class MemoryStorage {
  constructor() {
    this.map = new Map();
  }
  getItem(key) {
    return this.map.has(key) ? this.map.get(key) : null;
  }
  setItem(key, value) {
    this.map.set(key, String(value));
  }
  removeItem(key) {
    this.map.delete(key);
  }
  clear() {
    this.map.clear();
  }
}

const locationSelect = () => screen.getByRole('combobox', { name: 'Location' });

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: new MemoryStorage(),
    });
  });

  it('auto-selects the first location and navigates to a file', async () => {
    getLocations.mockResolvedValue(['TestDir', 'Other']);
    getFiles
      .mockResolvedValueOnce([{ filename: 'folder', type: 'directory' }])
      .mockResolvedValueOnce([{ filename: 'data.tmp', type: 'file' }]);

    render(<App />);

    await waitFor(() => expect(locationSelect()).toHaveValue('TestDir'));
    fireEvent.click(await screen.findByRole('button', { name: 'folder' }));

    await waitFor(() => {
      expect(getFiles).toHaveBeenLastCalledWith(
        'TestDir',
        'folder',
        expect.objectContaining({ signal: expect.any(AbortSignal) })
      );
    });

    fireEvent.click(await screen.findByRole('button', { name: 'data.tmp' }));

    expect(getFileUrl).toHaveBeenCalledWith('folder/data.tmp', 'fs', 'TestDir');
    expect(getFileUrl).toHaveBeenCalledWith('folder/data.tmp', 'hdr', 'TestDir');
    expect((await screen.findByTestId('raw-href')).textContent).toBe(
      '/mock/fs/TestDir/folder/data.tmp'
    );
    expect(screen.getByTestId('sds-href').textContent).toBe('/mock/hdr/TestDir/folder/data.tmp');
    expect(screen.getByTestId('file-name').textContent).toBe('folder/data.tmp');
  });

  it('ignores stale file fetch responses after switching locations', async () => {
    getLocations.mockResolvedValue(['A', 'B']);
    let resolveA;
    const slowA = new Promise((resolve) => {
      resolveA = resolve;
    });
    getFiles.mockImplementation((location) => {
      if (location === 'A') return slowA;
      return Promise.resolve([{ filename: 'b.tmp', type: 'file' }]);
    });

    render(<App />);

    await waitFor(() => expect(getFiles).toHaveBeenCalledWith('A', '', expect.any(Object)));
    fireEvent.change(locationSelect(), { target: { value: 'B' } });

    expect(await screen.findByRole('button', { name: 'b.tmp' })).toBeTruthy();
    resolveA([{ filename: 'old-a.tmp', type: 'file' }]);

    await waitFor(() => {
      expect(screen.queryByText('old-a.tmp')).toBeNull();
      expect(screen.getByText('b.tmp')).toBeTruthy();
    });
  });

  it('shows a retryable locations error', async () => {
    getLocations.mockRejectedValueOnce(new Error('network')).mockResolvedValueOnce(['TestDir']);
    getFiles.mockResolvedValue([]);

    render(<App />);

    expect(await screen.findByText('Failed to load locations.')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => expect(locationSelect()).toHaveValue('TestDir'));
    expect(getLocations).toHaveBeenCalledTimes(2);
  });

  it('places the file browser in a sidebar beside an empty plot area', async () => {
    getLocations.mockResolvedValue(['TestDir']);
    getFiles.mockResolvedValue([]);
    render(<App />);

    const sidebar = screen.getByRole('complementary', { name: 'File browser' });
    expect(within(sidebar).getByRole('combobox', { name: 'Location' })).toBeTruthy();
    expect(await within(sidebar).findByText('No files found')).toBeTruthy();

    const main = screen.getByRole('main');
    expect(within(main).getByRole('status')).toHaveTextContent('No file selected');
    expect(within(main).getByRole('status')).toHaveTextContent('Select a file from the sidebar');
  });

  it('highlights the selected file and replaces the empty state with plots', async () => {
    getLocations.mockResolvedValue(['TestDir']);
    getFiles.mockResolvedValue([
      { filename: 'a.tmp', type: 'file' },
      { filename: 'b.tmp', type: 'file' },
    ]);
    render(<App />);

    expect(screen.queryByTestId('raw-href')).toBeNull();
    fireEvent.click(await screen.findByRole('button', { name: 'b.tmp' }));

    expect(await screen.findByTestId('file-name')).toHaveTextContent('b.tmp');
    expect(screen.queryByText('No file selected')).toBeNull();
    expect(screen.getByRole('button', { name: 'b.tmp' })).toHaveAttribute('aria-current', 'true');
    expect(screen.getByRole('button', { name: 'a.tmp' })).not.toHaveAttribute('aria-current');
  });

  it('clears the selected file when switching locations', async () => {
    getLocations.mockResolvedValue(['A', 'B']);
    getFiles.mockResolvedValue([{ filename: 'x.tmp', type: 'file' }]);
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: 'x.tmp' }));
    expect(await screen.findByTestId('file-name')).toHaveTextContent('x.tmp');

    fireEvent.change(locationSelect(), { target: { value: 'B' } });

    expect(await screen.findByText('No file selected')).toBeTruthy();
    expect(screen.queryByTestId('raw-href')).toBeNull();
  });

  it('navigates back to the location root via breadcrumbs', async () => {
    getLocations.mockResolvedValue(['TestDir']);
    getFiles.mockResolvedValue([{ filename: 'folder', type: 'directory' }]);
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: 'folder' }));
    await waitFor(() =>
      expect(getFiles).toHaveBeenLastCalledWith('TestDir', 'folder', expect.any(Object))
    );

    const crumbs = within(screen.getByRole('navigation', { name: 'Current folder' }));
    fireEvent.click(crumbs.getByRole('button', { name: 'TestDir' }));

    await waitFor(() =>
      expect(getFiles).toHaveBeenLastCalledWith('TestDir', '', expect.any(Object))
    );
  });

  it('collapses the sidebar and remembers the choice', async () => {
    getLocations.mockResolvedValue([]);
    const { unmount } = render(<App />);
    expect(await screen.findByText('No locations found')).toBeTruthy();

    const toggle = screen.getByRole('button', { name: 'Hide file browser' });
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    fireEvent.click(toggle);

    expect(screen.queryByRole('complementary', { name: 'File browser' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Show file browser' })).toHaveAttribute(
      'aria-expanded',
      'false'
    );

    unmount();
    render(<App />);
    expect(await screen.findByRole('button', { name: 'Show file browser' })).toBeTruthy();
  });

  it('switches between stacked and side-by-side plot layouts', async () => {
    getLocations.mockResolvedValue([]);
    render(<App />);
    expect(await screen.findByText('No locations found')).toBeTruthy();

    const layout = within(screen.getByRole('group', { name: 'Plot layout' }));
    expect(layout.getByRole('button', { name: 'Stacked' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('main')).toHaveClass('plots-stacked');

    fireEvent.click(layout.getByRole('button', { name: 'Side by side' }));

    expect(layout.getByRole('button', { name: 'Side by side' })).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    expect(screen.getByRole('main')).toHaveClass('plots-columns');
    expect(JSON.parse(window.localStorage.getItem('sds.plotLayout'))).toBe('columns');
  });
});
