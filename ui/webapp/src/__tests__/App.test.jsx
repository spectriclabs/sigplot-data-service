import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen, waitFor, within } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import App from '../App';
import { getLocations, getFileUrl } from '../api/sds';

vi.mock('../api/sds', async (importOriginal) => ({
  ...(await importOriginal()),
  getLocations: vi.fn(),
  getFileUrl: vi.fn((file, mode, location) => `/mock/${mode}/${location}/${file}`),
}));

// The real tree is covered in DataTree.test; here it only needs to report picks.
vi.mock('../components/DataTree', () => ({
  default: ({ locations, onSelectFile }) => (
    <ul aria-label="Mock tree">
      {locations.map((location) => (
        <li key={location}>
          <button type="button" onClick={() => onSelectFile({ location, path: 'folder/data.tmp' })}>
            {location}/folder/data.tmp
          </button>
        </li>
      ))}
    </ul>
  ),
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

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: new MemoryStorage(),
    });
  });

  it('plots the file picked in the tree', async () => {
    getLocations.mockResolvedValue(['TestDir', 'Other']);
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: 'Other/folder/data.tmp' }));

    expect(getFileUrl).toHaveBeenCalledWith('folder/data.tmp', 'fs', 'Other');
    expect(getFileUrl).toHaveBeenCalledWith('folder/data.tmp', 'hdr', 'Other');
    expect((await screen.findByTestId('raw-href')).textContent).toBe(
      '/mock/fs/Other/folder/data.tmp'
    );
    expect(screen.getByTestId('sds-href').textContent).toBe('/mock/hdr/Other/folder/data.tmp');
    expect(screen.getByTestId('file-name').textContent).toBe('Other/folder/data.tmp');
    expect(screen.queryByText('No file selected')).toBeNull();
  });

  it('places the data tree in a sidebar beside an empty plot area', async () => {
    getLocations.mockResolvedValue(['TestDir']);
    render(<App />);

    const sidebar = screen.getByRole('complementary', { name: 'File browser' });
    expect(await within(sidebar).findByRole('list', { name: 'Mock tree' })).toBeTruthy();

    const status = within(screen.getByRole('main')).getByRole('status');
    expect(status).toHaveTextContent('No file selected');
    expect(status).toHaveTextContent('Open a location in the sidebar');
  });

  it('shows a retryable locations error', async () => {
    getLocations.mockRejectedValueOnce(new Error('network')).mockResolvedValueOnce(['TestDir']);
    render(<App />);

    expect(await screen.findByText('Failed to load locations.')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    expect(await screen.findByRole('button', { name: 'TestDir/folder/data.tmp' })).toBeTruthy();
    expect(getLocations).toHaveBeenCalledTimes(2);
  });

  it('collapses the sidebar and remembers the choice', async () => {
    getLocations.mockResolvedValue([]);
    const { unmount } = render(<App />);
    expect(await screen.findByText('No locations configured')).toBeTruthy();

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
    expect(await screen.findByText('No locations configured')).toBeTruthy();

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
