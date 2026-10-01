import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import FileBrowser from '../components/FileBrowser';

vi.mock('../components/DataTree', () => ({
  default: ({ locations, onSelectFile }) => (
    <div data-testid="data-tree">
      {locations.join(',')}
      <button type="button" onClick={() => onSelectFile({ location: 'A', path: 'x.tmp' })}>
        pick
      </button>
    </div>
  ),
}));

const defaultProps = {
  locations: ['A', 'B'],
  locationsStatus: 'success',
  onRetryLocations: vi.fn(),
  onSelectFile: vi.fn(),
};

describe('FileBrowser', () => {
  it('renders the data tree with every location', async () => {
    const onSelectFile = vi.fn();
    render(<FileBrowser {...defaultProps} onSelectFile={onSelectFile} />);

    expect(await screen.findByTestId('data-tree')).toHaveTextContent('A,B');
    fireEvent.click(screen.getByRole('button', { name: 'pick' }));
    expect(onSelectFile).toHaveBeenCalledWith({ location: 'A', path: 'x.tmp' });
  });

  it('shows a loading state while locations load', () => {
    render(<FileBrowser {...defaultProps} locations={[]} locationsStatus="loading" />);
    expect(screen.getByText('Loading locations...')).toBeTruthy();
  });

  it('shows a retryable error when locations fail', () => {
    const onRetry = vi.fn();
    render(
      <FileBrowser {...defaultProps} locations={[]} locationsStatus="error" onRetryLocations={onRetry} />
    );
    expect(screen.getByRole('alert')).toHaveTextContent('Failed to load locations.');
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(onRetry).toHaveBeenCalled();
  });

  it('explains when no locations are configured', () => {
    render(<FileBrowser {...defaultProps} locations={[]} locationsStatus="success" />);
    expect(screen.getByText('No locations configured')).toBeTruthy();
    expect(screen.queryByTestId('data-tree')).toBeNull();
  });

  it('does not crash when locations is null', () => {
    render(<FileBrowser {...defaultProps} locations={null} locationsStatus="success" />);
    expect(screen.getByText('No locations configured')).toBeTruthy();
  });
});
