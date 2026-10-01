import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { Plot } from 'sigplot';
import SigPlotViewer from '../components/SigPlotViewer';

const instances = [];
const callbacks = expect.objectContaining({
  onload: expect.any(Function),
  onerror: expect.any(Function),
});

vi.mock('sigplot', () => ({
  Plot: vi.fn().mockImplementation(() => {
    const instance = {
      deoverlay: vi.fn(),
      remove_layer: vi.fn(),
      checkresize: vi.fn(),
      change_settings: vi.fn(),
      mimic: vi.fn(),
      hide_spinner: vi.fn(),
      unmimic: vi.fn(),
      disable_listeners: vi.fn(),
      overlay_href: vi.fn((href, callbacks) => {
        const layer = `layer:${href}`;
        instance.loads.push({ href, ...callbacks, layer });
        return layer;
      }),
      loads: [],
    };
    instances.push(instance);
    return instance;
  }),
}));

// jsdom has no layout or ResizeObserver; fake both so tests control sizing.
const observers = [];
class FakeResizeObserver {
  constructor(callback) {
    this.callback = callback;
    this.targets = new Set();
    this.disconnect = vi.fn(() => this.targets.clear());
    observers.push(this);
  }
  observe(el) {
    this.targets.add(el);
  }
  unobserve(el) {
    this.targets.delete(el);
  }
}

let size = { width: 800, height: 400 };

function triggerResize() {
  act(() => {
    observers.forEach((o) => {
      if (o.targets.size) o.callback([...o.targets].map((target) => ({ target })), o);
    });
  });
}

describe('SigPlotViewer', () => {
  beforeEach(() => {
    instances.length = 0;
    observers.length = 0;
    Plot.mockClear();
    size = { width: 800, height: 400 };
    vi.stubGlobal('ResizeObserver', FakeResizeObserver);
    for (const [prop, key] of [['clientWidth', 'width'], ['clientHeight', 'height']]) {
      Object.defineProperty(HTMLElement.prototype, prop, {
        configurable: true,
        get() {
          return size[key];
        },
      });
    }
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    // Drop the overrides so jsdom's Element.prototype getters apply again.
    delete HTMLElement.prototype.clientWidth;
    delete HTMLElement.prototype.clientHeight;
  });

  it('removes stale layers when an older async load finishes late', () => {
    const { rerender } = render(<SigPlotViewer rawHref="/first.tmp" sdsHref={null} />);
    const rawPlot = instances[0];

    rerender(<SigPlotViewer rawHref="/second.tmp" sdsHref={null} />);
    act(() => {
      rawPlot.loads[0].onload();
      rawPlot.loads[1].onload();
    });

    expect(rawPlot.remove_layer).toHaveBeenCalledWith('layer:/first.tmp');
    expect(rawPlot.remove_layer).not.toHaveBeenCalledWith('layer:/second.tmp');
  });

  it('scales raw rasters over every line and uses the SDS layer for the tiled panel', () => {
    render(<SigPlotViewer rawHref="/raw.tmp" sdsHref="/sds.tmp" />);

    expect(instances[0].overlay_href).toHaveBeenCalledWith('/raw.tmp', callbacks, {
      lpb: Infinity,
    });
    expect(instances[1].overlay_href).toHaveBeenCalledWith('/sds.tmp', callbacks, {
      layerType: 'SDS',
    });
  });

  it('deoverlays, detaches listeners, and stops observing on unmount', () => {
    const { unmount } = render(<SigPlotViewer rawHref="/file.tmp" sdsHref="/file.hdr" />);

    unmount();

    instances.forEach((instance) => {
      expect(instance.deoverlay).toHaveBeenCalled();
      expect(instance.disable_listeners).toHaveBeenCalled();
    });
    observers.forEach((observer) => expect(observer.disconnect).toHaveBeenCalled());
  });

  it('shows the selected file name in each panel header', () => {
    render(<SigPlotViewer rawHref="/a" sdsHref="/b" fileName="sub/mydata.tmp" />);

    expect(screen.getByRole('region', { name: 'Raw file' }).textContent).toContain('sub/mydata.tmp');
    expect(screen.getByRole('region', { name: 'SDS tiled' }).textContent).toContain(
      'sub/mydata.tmp'
    );
  });

  it('observes each plot container and calls checkresize on resize', () => {
    render(<SigPlotViewer rawHref="/a" sdsHref="/b" />);
    const containers = screen.getAllByTestId('plot-canvas');

    expect(observers).toHaveLength(2);
    observers.forEach((observer, i) => expect(observer.targets.has(containers[i])).toBe(true));
    instances.forEach((instance) => expect(instance.checkresize).not.toHaveBeenCalled());

    size = { width: 1200, height: 500 };
    triggerResize();

    instances.forEach((instance) => expect(instance.checkresize).toHaveBeenCalledTimes(1));
  });

  it('waits for a non-zero container size before creating plots', () => {
    size = { width: 0, height: 0 };
    render(<SigPlotViewer rawHref="/a.tmp" sdsHref="/b.tmp" />);

    expect(Plot).not.toHaveBeenCalled();

    size = { width: 900, height: 300 };
    triggerResize();

    expect(Plot).toHaveBeenCalledTimes(2);
    expect(instances[0].overlay_href).toHaveBeenCalledWith('/a.tmp', callbacks, {
      lpb: Infinity,
    });
    expect(instances[1].overlay_href).toHaveBeenCalledWith('/b.tmp', callbacks, {
      layerType: 'SDS',
    });
    instances.forEach((instance) => expect(instance.checkresize).not.toHaveBeenCalled());
  });

  it('skips checkresize while the container is collapsed to zero size', () => {
    render(<SigPlotViewer rawHref="/a" sdsHref="/b" />);

    size = { width: 0, height: 0 };
    triggerResize();

    instances.forEach((instance) => expect(instance.checkresize).not.toHaveBeenCalled());
  });

  it('shows a loading indicator until the layer loads', () => {
    render(<SigPlotViewer rawHref="/a.tmp" sdsHref={null} />);
    const raw = screen.getByRole('region', { name: 'Raw file' });

    expect(raw).toHaveAttribute('aria-busy', 'true');
    expect(raw).toHaveTextContent('Loading');

    act(() => instances[0].loads[0].onload());

    expect(raw).toHaveAttribute('aria-busy', 'false');
    expect(raw).not.toHaveTextContent('Loading');
  });

  it('shows an error when the current layer fails to load', () => {
    render(<SigPlotViewer rawHref="/a.tmp" sdsHref={null} />);

    act(() => instances[0].loads[0].onerror('Failed to load data'));

    expect(screen.getByRole('alert')).toHaveTextContent('Couldn’t load this file.');
    expect(instances[0].hide_spinner).toHaveBeenCalledWith(true);
  });

  it('ignores errors from a superseded load', () => {
    const { rerender } = render(<SigPlotViewer rawHref="/old.tmp" sdsHref={null} />);
    rerender(<SigPlotViewer rawHref="/new.tmp" sdsHref={null} />);

    act(() => instances[0].loads[0].onerror('stale'));

    expect(screen.queryByRole('alert')).toBeNull();
    expect(instances[0].hide_spinner).not.toHaveBeenCalled();
    expect(screen.getByRole('region', { name: 'Raw file' })).toHaveAttribute('aria-busy', 'true');
  });

  it('clears the plot note before loading the next file', () => {
    const { rerender } = render(<SigPlotViewer rawHref="/first.tmp" sdsHref={null} />);
    const raw = instances[0];
    raw.change_settings.mockClear();

    rerender(<SigPlotViewer rawHref="/second.tmp" sdsHref={null} />);

    expect(raw.change_settings).toHaveBeenCalledWith({ note: '' });
    expect(raw.deoverlay.mock.invocationCallOrder.at(-1)).toBeLessThan(
      raw.change_settings.mock.invocationCallOrder.at(-1)
    );
    expect(raw.change_settings.mock.invocationCallOrder.at(-1)).toBeLessThan(
      raw.overlay_href.mock.invocationCallOrder.at(-1)
    );
  });

  it('mirrors zoom, unzoom and pan between the two plots', () => {
    const { unmount } = render(<SigPlotViewer rawHref="/a.tmp" sdsHref="/a.hdr" />);
    const [raw, sds] = instances;
    const mask = { zoom: true, unzoom: true, pan: true };

    expect(raw.mimic).toHaveBeenCalledWith(sds, mask);
    expect(sds.mimic).toHaveBeenCalledWith(raw, mask);

    unmount();

    expect(raw.unmimic).toHaveBeenCalled();
    expect(sds.unmimic).toHaveBeenCalled();
  });

  it('links the plots only once both exist', () => {
    size = { width: 0, height: 0 };
    render(<SigPlotViewer rawHref="/a.tmp" sdsHref="/a.hdr" />);
    expect(instances).toHaveLength(0);

    size = { width: 900, height: 300 };
    triggerResize();

    const [raw, sds] = instances;
    expect(raw.mimic).toHaveBeenCalledTimes(1);
    expect(sds.mimic).toHaveBeenCalledTimes(1);
  });
});
