import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen, waitFor } from '@testing-library/react';
import App from '../App';
import { getLocations, getFiles, getFileUrl } from '../api/sds';

vi.mock('../api/sds', () => ({
  getLocations: vi.fn(),
  getFiles: vi.fn(),
  getFileUrl: vi.fn((file, mode, location) => `/mock/${mode}/${location}/${file}`),
}));

vi.mock('../components/SigPlotViewer', () => ({
  default: ({ rawHref, sdsHref }) => (
    <div>
      <output data-testid="raw-href">{rawHref || ''}</output>
      <output data-testid="sds-href">{sdsHref || ''}</output>
    </div>
  ),
}));

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('navigates from location to folder to file', async () => {
    getLocations.mockResolvedValue(['TestDir']);
    getFiles
      .mockResolvedValueOnce([{ filename: 'folder', type: 'directory' }])
      .mockResolvedValueOnce([{ filename: 'data.tmp', type: 'file' }]);

    render(<App />);

    fireEvent.click(await screen.findByText('TestDir'));
    fireEvent.click(await screen.findByText('folder'));

    await waitFor(() => {
      expect(getFiles).toHaveBeenLastCalledWith(
        'TestDir',
        'folder',
        expect.objectContaining({ signal: expect.any(AbortSignal) })
      );
    });

    fireEvent.click(await screen.findByText('data.tmp'));

    expect(getFileUrl).toHaveBeenCalledWith('folder/data.tmp', 'fs', 'TestDir');
    expect(getFileUrl).toHaveBeenCalledWith('folder/data.tmp', 'hdr', 'TestDir');
    expect((await screen.findByTestId('raw-href')).textContent).toBe(
      '/mock/fs/TestDir/folder/data.tmp'
    );
    expect(screen.getByTestId('sds-href').textContent).toBe(
      '/mock/hdr/TestDir/folder/data.tmp'
    );
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

    fireEvent.click(await screen.findByText('A'));
    await waitFor(() => expect(getFiles).toHaveBeenCalledWith('A', '', expect.any(Object)));
    fireEvent.click(screen.getByText('B'));

    expect(await screen.findByText('b.tmp')).toBeTruthy();
    resolveA([{ filename: 'old-a.tmp', type: 'file' }]);

    await waitFor(() => {
      expect(screen.queryByText('old-a.tmp')).toBeNull();
      expect(screen.getByText('b.tmp')).toBeTruthy();
    });
  });

  it('shows a retryable locations error', async () => {
    getLocations.mockRejectedValueOnce(new Error('network'));

    render(<App />);

    expect(await screen.findByText('Failed to load locations.')).toBeTruthy();
    expect(screen.getByText('Retry')).toBeTruthy();
  });
});
