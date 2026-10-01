import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, screen, within } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import FileBrowser from '../components/FileBrowser';

const defaultProps = {
  locations: ['TestDir', 'minio'],
  locationsStatus: 'success',
  selectedLocation: '',
  onSelectLocation: vi.fn(),
  onRetryLocations: vi.fn(),
  files: [],
  filesStatus: 'success',
  onSelectFile: vi.fn(),
  selectedFile: null,
  path: '',
  onGoBack: vi.fn(),
  onNavigate: vi.fn(),
};

const files = [
  { filename: 'b.tmp', type: 'file' },
  { filename: 'zdir', type: 'directory' },
  { filename: 'a10.tmp', type: 'file' },
  { filename: 'a2.tmp', type: 'file' },
];

function renderAt(props = {}) {
  return render(
    <FileBrowser {...defaultProps} selectedLocation="TestDir" files={files} {...props} />
  );
}

const rowNames = () =>
  within(screen.getByRole('list', { name: 'Files' }))
    .getAllByRole('button')
    .map((b) => b.textContent);

describe('FileBrowser', () => {
  it('offers locations in a labelled dropdown', () => {
    const onSelect = vi.fn();
    render(<FileBrowser {...defaultProps} onSelectLocation={onSelect} />);

    const select = screen.getByRole('combobox', { name: 'Location' });
    expect(within(select).getByRole('option', { name: 'TestDir' })).toBeTruthy();
    expect(within(select).getByRole('option', { name: 'minio' })).toBeTruthy();

    fireEvent.change(select, { target: { value: 'minio' } });
    expect(onSelect).toHaveBeenCalledWith('minio');
  });

  it('reflects the selected location in the dropdown', () => {
    render(<FileBrowser {...defaultProps} selectedLocation="minio" />);
    expect(screen.getByRole('combobox', { name: 'Location' })).toHaveValue('minio');
  });

  it('hides the file area until a location is chosen', () => {
    render(<FileBrowser {...defaultProps} />);
    expect(screen.queryByRole('textbox', { name: 'Filter files' })).toBeNull();
    expect(screen.queryByRole('navigation', { name: 'Current folder' })).toBeNull();
  });

  it('lists folders first, then files in natural order', () => {
    renderAt();
    expect(rowNames()).toEqual(['zdir', 'a2.tmp', 'a10.tmp', 'b.tmp']);
    expect(screen.getByText('4 items')).toBeTruthy();
  });

  it('calls onSelectFile when a row is clicked', () => {
    const onSelect = vi.fn();
    renderAt({ onSelectFile: onSelect });
    fireEvent.click(screen.getByRole('button', { name: 'b.tmp' }));
    expect(onSelect).toHaveBeenCalledWith({ filename: 'b.tmp', type: 'file' });
  });

  it('highlights the selected file only in its own folder', () => {
    const { rerender } = renderAt({ path: 'sub', selectedFile: 'sub/b.tmp' });
    const selected = screen.getByRole('button', { name: 'b.tmp' });
    expect(selected).toHaveAttribute('aria-current', 'true');
    expect(selected).toHaveClass('is-selected');
    expect(screen.getByRole('button', { name: 'a2.tmp' })).not.toHaveAttribute('aria-current');

    rerender(
      <FileBrowser
        {...defaultProps}
        selectedLocation="TestDir"
        files={files}
        path=""
        selectedFile="sub/b.tmp"
      />
    );
    expect(screen.getByRole('button', { name: 'b.tmp' })).not.toHaveAttribute('aria-current');
  });

  it('filters the list and reports matches', () => {
    renderAt();
    fireEvent.change(screen.getByRole('searchbox', { name: 'Filter files' }), {
      target: { value: 'A' },
    });
    expect(rowNames()).toEqual(['a2.tmp', 'a10.tmp']);
    expect(screen.getByText('2 of 4 items')).toBeTruthy();

    fireEvent.change(screen.getByRole('searchbox', { name: 'Filter files' }), {
      target: { value: 'nope' },
    });
    expect(screen.getByText('No files match “nope”')).toBeTruthy();
  });

  it('clears the filter when the folder changes', () => {
    const { rerender } = renderAt();
    const search = screen.getByRole('searchbox', { name: 'Filter files' });
    fireEvent.change(search, { target: { value: 'a' } });

    rerender(
      <FileBrowser {...defaultProps} selectedLocation="TestDir" files={files} path="zdir" />
    );
    expect(screen.getByRole('searchbox', { name: 'Filter files' })).toHaveValue('');
  });

  it('keeps folder icons out of the accessible name', () => {
    renderAt();
    expect(screen.getByRole('button', { name: 'zdir' })).toBeTruthy();
  });

  it('enables the up button only inside a folder', () => {
    const onGoBack = vi.fn();
    const { rerender } = renderAt({ onGoBack });
    expect(screen.getByRole('button', { name: 'Up one folder' })).toBeDisabled();

    rerender(
      <FileBrowser
        {...defaultProps}
        selectedLocation="TestDir"
        files={files}
        path="zdir"
        onGoBack={onGoBack}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Up one folder' }));
    expect(onGoBack).toHaveBeenCalled();
  });

  it('renders breadcrumbs that navigate to ancestor folders', () => {
    const onNavigate = vi.fn();
    renderAt({ path: 'one/two', onNavigate });
    const crumbs = within(screen.getByRole('navigation', { name: 'Current folder' }));

    expect(crumbs.getByRole('button', { name: 'two' })).toHaveAttribute('aria-current', 'location');
    fireEvent.click(crumbs.getByRole('button', { name: 'one' }));
    expect(onNavigate).toHaveBeenLastCalledWith('one');
    fireEvent.click(crumbs.getByRole('button', { name: 'TestDir' }));
    expect(onNavigate).toHaveBeenLastCalledWith('');
  });

  it('does not crash when files is null', () => {
    renderAt({ files: null });
    expect(screen.getByText('No files found')).toBeTruthy();
  });

  it('distinguishes locations errors from loading', () => {
    const onRetry = vi.fn();
    render(
      <FileBrowser
        {...defaultProps}
        locations={[]}
        locationsStatus="error"
        onRetryLocations={onRetry}
      />
    );
    expect(screen.getByText('Failed to load locations.')).toBeTruthy();
    expect(screen.queryByText('Loading locations...')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(onRetry).toHaveBeenCalled();
  });

  it('shows a disabled dropdown while locations load', () => {
    render(<FileBrowser {...defaultProps} locations={[]} locationsStatus="loading" />);
    expect(screen.getByRole('combobox', { name: 'Location' })).toBeDisabled();
    expect(screen.getByText('Loading locations...')).toBeTruthy();
  });

  it('distinguishes file errors from empty folders', () => {
    renderAt({ filesStatus: 'error' });
    expect(screen.getByText('Failed to load files.')).toBeTruthy();
    expect(screen.queryByText('No files found')).toBeNull();
  });

  it('only lists folders and BLUE files', () => {
    renderAt({
      files: [
        { filename: 'main.go', type: 'file' },
        { filename: 'data.tmp', type: 'file' },
        { filename: 'config.json', type: 'file' },
        { filename: 'src', type: 'directory' },
        { filename: 'p.prm', type: 'file' },
      ],
    });
    expect(rowNames()).toEqual(['src', 'data.tmp', 'p.prm']);
    expect(screen.getByText('3 items')).toBeTruthy();
  });
});
