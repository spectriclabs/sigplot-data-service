import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import SigPlotViewer from '../components/SigPlotViewer';

const instances = [];

vi.mock('sigplot', () => ({
  Plot: vi.fn().mockImplementation(() => {
    const instance = {
      deoverlay: vi.fn(),
      remove_layer: vi.fn(),
      overlay_href: vi.fn((href, onload) => {
        const layer = `layer:${href}`;
        instance.loads.push({ href, onload, layer });
        return layer;
      }),
      loads: [],
    };
    instances.push(instance);
    return instance;
  }),
}));

describe('SigPlotViewer', () => {
  beforeEach(() => {
    instances.length = 0;
  });

  it('removes stale layers when an older async load finishes late', () => {
    const { rerender } = render(<SigPlotViewer rawHref="/first.tmp" sdsHref={null} />);
    const rawPlot = instances[0];

    rerender(<SigPlotViewer rawHref="/second.tmp" sdsHref={null} />);
    rawPlot.loads[0].onload();
    rawPlot.loads[1].onload();

    expect(rawPlot.remove_layer).toHaveBeenCalledWith('layer:/first.tmp');
    expect(rawPlot.remove_layer).not.toHaveBeenCalledWith('layer:/second.tmp');
  });

  it('deoverlays and empties plots on unmount', () => {
    const { unmount } = render(<SigPlotViewer rawHref="/file.tmp" sdsHref="/file.hdr" />);

    unmount();

    expect(instances[0].deoverlay).toHaveBeenCalled();
    expect(instances[1].deoverlay).toHaveBeenCalled();
  });
});
