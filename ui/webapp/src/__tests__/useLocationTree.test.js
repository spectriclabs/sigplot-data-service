import { describe, it, expect, vi, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { FileTree } from '@pierre/trees';
import { splitTreePath, toTreePath, useLocationTree } from '../hooks/useLocationTree';

const dir = (filename) => ({ filename, type: 'directory' });
const file = (filename) => ({ filename, type: 'file' });

const models = [];
function makeModel() {
  const model = new FileTree({ paths: [] });
  models.push(model);
  return model;
}

const visiblePaths = (model) =>
  model.getVisibleRows(0, model.getVisibleCount() - 1).map((row) => row.path);

describe('tree path helpers', () => {
  it('builds canonical Trees paths with trailing slashes for folders', () => {
    expect(toTreePath('', file('a.tmp'))).toBe('a.tmp');
    expect(toTreePath('Loc/', dir('sub'))).toBe('Loc/sub/');
    expect(toTreePath('Loc/sub/', file('b.tmp'))).toBe('Loc/sub/b.tmp');
  });

  it('splits tree paths into a location and a path inside it', () => {
    const locations = ['A', 'B/C'];
    expect(splitTreePath('A/x/y.tmp', locations)).toEqual({ location: 'A', path: 'x/y.tmp' });
    expect(splitTreePath('B/C/z.tmp', locations)).toEqual({ location: 'B/C', path: 'z.tmp' });
    expect(splitTreePath('A/', locations)).toEqual({ location: 'A', path: '' });
    expect(splitTreePath('Other/z.tmp', locations)).toBeNull();
  });
});

describe('useLocationTree', () => {
  afterEach(() => {
    models.splice(0).forEach((model) => model.cleanUp());
  });

  it('shows locations as folders and loads the first one', async () => {
    const model = makeModel();
    const locations = ['First', 'Second'];
    const list = vi.fn().mockResolvedValue([file('b.tmp'), dir('sub'), file('a.prm')]);

    renderHook(() => useLocationTree(model, locations, list));

    await waitFor(() =>
      expect(visiblePaths(model)).toEqual([
        'First/',
        'First/sub/',
        'First/a.prm',
        'First/b.tmp',
        'Second/',
      ])
    );
    expect(list).toHaveBeenCalledTimes(1);
    expect(list).toHaveBeenCalledWith('First', '', { signal: expect.any(AbortSignal) });
  });

  it('only shows folders and BLUE files', async () => {
    const model = makeModel();
    const locations = ['Loc'];
    const list = vi
      .fn()
      .mockResolvedValue([file('main.go'), file('data.tmp'), file('notes.txt'), dir('src'), file('p.prm')]);

    renderHook(() => useLocationTree(model, locations, list));

    await waitFor(() =>
      expect(visiblePaths(model)).toEqual(['Loc/', 'Loc/src/', 'Loc/data.tmp', 'Loc/p.prm'])
    );
  });

  it('fetches locations and folders only when first expanded', async () => {
    const model = makeModel();
    const locations = ['A', 'B'];
    const list = vi.fn(async (location, path) => {
      if (location === 'A') return [file('a.tmp')];
      if (path === '') return [dir('sub')];
      return [file('deep.tmp')];
    });
    renderHook(() => useLocationTree(model, locations, list));
    await waitFor(() => expect(list).toHaveBeenCalledTimes(1));

    act(() => model.getItem('B/').expand());
    await waitFor(() => expect(model.getItem('B/sub/')).not.toBeNull());
    expect(list).toHaveBeenLastCalledWith('B', '', expect.any(Object));

    act(() => model.getItem('B/sub/').expand());
    await waitFor(() => expect(model.getItem('B/sub/deep.tmp')).not.toBeNull());
    expect(list).toHaveBeenLastCalledWith('B', 'sub', expect.any(Object));

    act(() => {
      model.getItem('B/sub/').collapse();
      model.getItem('B/sub/').expand();
    });
    expect(list).toHaveBeenCalledTimes(3);
  });

  it('collapses a folder that fails to load so expanding retries it', async () => {
    const model = makeModel();
    const locations = ['Loc'];
    const list = vi.fn(async (_location, path) => {
      if (path === '') return [dir('sub')];
      throw new Error('boom');
    });
    const { result } = renderHook(() => useLocationTree(model, locations, list));
    await waitFor(() => expect(model.getItem('Loc/sub/')).not.toBeNull());

    act(() => model.getItem('Loc/sub/').expand());

    await waitFor(() => expect(result.current.folderError).toBe('Loc/sub/'));
    expect(model.getItem('Loc/sub/').isExpanded()).toBe(false);

    list.mockImplementation(async () => [file('ok.tmp')]);
    act(() => model.getItem('Loc/sub/').expand());

    await waitFor(() => expect(model.getItem('Loc/sub/ok.tmp')).not.toBeNull());
    expect(result.current.folderError).toBeNull();
  });

  it('reports a location that fails to load', async () => {
    const model = makeModel();
    const locations = ['Down'];
    const list = vi.fn().mockRejectedValue(new Error('offline'));
    const { result } = renderHook(() => useLocationTree(model, locations, list));

    await waitFor(() => expect(result.current.folderError).toBe('Down/'));
    expect(model.getItem('Down/').isExpanded()).toBe(false);
  });

  it('aborts in-flight requests and ignores late results after unmount', async () => {
    const model = makeModel();
    const locations = ['Loc'];
    let resolve;
    let signal;
    const list = vi.fn((_location, _path, options) => {
      signal = options.signal;
      return new Promise((res) => {
        resolve = res;
      });
    });
    const { unmount } = renderHook(() => useLocationTree(model, locations, list));
    await waitFor(() => expect(list).toHaveBeenCalled());

    unmount();
    expect(signal.aborted).toBe(true);

    await act(async () => resolve([file('late.tmp')]));
    expect(model.getItem('Loc/late.tmp')).toBeNull();
  });

  it('rebuilds the tree when the locations change', async () => {
    const model = makeModel();
    const list = vi.fn().mockResolvedValue([file('x.tmp')]);
    const { rerender } = renderHook(({ locations }) => useLocationTree(model, locations, list), {
      initialProps: { locations: ['Old'] },
    });
    await waitFor(() => expect(model.getItem('Old/x.tmp')).not.toBeNull());

    rerender({ locations: ['New'] });

    await waitFor(() => expect(visiblePaths(model)).toEqual(['New/', 'New/x.tmp']));
    expect(model.getItem('Old/')).toBeNull();
  });
});
