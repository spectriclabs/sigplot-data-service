import React, { useRef, useEffect, useState } from 'react';
import { Plot } from 'sigplot';
import { AlertIcon } from './icons';

const PLOT_OPTIONS = {
  all: true,
  expand: true,
  autohide_panbars: true,
};

// SigPlot color-scales a raster from only its first 16 lines by default, so
// a file that starts with a constant band (like mydata_SB_600_600.tmp) comes
// out a single color. Scanning every line uses the whole file's z range,
// which is what SDS does server-side for the tiled view.
const RAW_LAYER_OPTIONS = { lpb: Infinity };
const SDS_LAYER_OPTIONS = { layerType: 'SDS' };

// Both panels show the same file on the same axes, so zooming, unzooming or
// panning one plot is mirrored on the other.
const MIRROR = { zoom: true, unzoom: true, pan: true };

function removeLayer(plot, layer) {
  if (!plot || layer == null) return;
  if (Array.isArray(layer)) {
    layer.forEach((entry) => removeLayer(plot, entry));
    return;
  }
  if (plot.remove_layer) plot.remove_layer(layer);
}

function hasSize(el) {
  return el.clientWidth > 0 && el.clientHeight > 0;
}

function SigPlotPanel({ href, layerOptions, title, fileName, onPlot }) {
  const containerRef = useRef(null);
  const latestHrefRef = useRef(null);
  const onPlotRef = useRef(onPlot);
  onPlotRef.current = onPlot;
  const [plot, setPlot] = useState(null);
  const [status, setStatus] = useState('idle');

  // Create the plot once its container has a real size (SigPlot sizes its
  // canvases from the container on construction), then keep it in sync with
  // layout changes via checkresize().
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return undefined;
    const canObserve = typeof ResizeObserver !== 'undefined';
    let instance = null;

    const sync = () => {
      if (instance) {
        if (hasSize(el)) instance.checkresize();
        return;
      }
      if (canObserve && !hasSize(el)) return;
      instance = new Plot(el, PLOT_OPTIONS);
      setPlot(instance);
      onPlotRef.current?.(instance);
    };

    sync();
    const observer = canObserve ? new ResizeObserver(sync) : null;
    if (observer) observer.observe(el);

    return () => {
      if (observer) observer.disconnect();
      latestHrefRef.current = null;
      if (instance) onPlotRef.current?.(null);
      if (instance) {
        if (instance.deoverlay) instance.deoverlay();
        if (instance.disable_listeners) instance.disable_listeners();
      }
      el.replaceChildren();
    };
  }, []);

  useEffect(() => {
    if (!plot) return undefined;
    latestHrefRef.current = href;
    plot.deoverlay();
    // SigPlot builds the plot note (the file name drawn in the corner) only
    // while it is empty and never clears it on deoverlay, so a reused plot
    // would keep showing the first file's name.
    if (plot.change_settings) plot.change_settings({ note: '' });
    if (!href) {
      setStatus('idle');
      return undefined;
    }
    setStatus('loading');
    const layer = plot.overlay_href(
      href,
      {
        onload: () => {
          if (latestHrefRef.current !== href) {
            removeLayer(plot, layer);
            return;
          }
          setStatus('ready');
        },
        onerror: () => {
          if (latestHrefRef.current !== href) return;
          setStatus('error');
          // SigPlot leaves its spinner running when an SDS request fails.
          if (plot.hide_spinner) plot.hide_spinner(true);
        },
      },
      layerOptions && { ...layerOptions }
    );
    return () => {
      if (latestHrefRef.current === href) latestHrefRef.current = null;
    };
  }, [plot, href, layerOptions]);

  return (
    <section className="plot-card" aria-label={title} aria-busy={status === 'loading'}>
      <header className="plot-card-header">
        <h2 className="plot-card-title">{title}</h2>
        {fileName && (
          <span className="plot-card-file" title={fileName}>
            {fileName}
          </span>
        )}
        {/* SigPlot draws its own spinner over the plot while loading, so the
            header only needs a text status. */}
        {status === 'loading' && <span className="plot-card-status">Loading…</span>}
      </header>
      <div className="plot-card-body">
        <div ref={containerRef} className="plot-canvas" data-testid="plot-canvas" />
        {status === 'error' && (
          <div className="plot-card-error" role="alert">
            <AlertIcon size={20} />
            <span>Couldn’t load this file.</span>
          </div>
        )}
      </div>
    </section>
  );
}

export default function SigPlotViewer({ rawHref, sdsHref, fileName }) {
  const [rawPlot, setRawPlot] = useState(null);
  const [sdsPlot, setSdsPlot] = useState(null);

  useEffect(() => {
    if (!rawPlot?.mimic || !sdsPlot?.mimic) return undefined;
    // SigPlot's zoom/pan guards stop the two listeners from echoing forever.
    rawPlot.mimic(sdsPlot, MIRROR);
    sdsPlot.mimic(rawPlot, MIRROR);
    return () => {
      rawPlot.unmimic();
      sdsPlot.unmimic();
    };
  }, [rawPlot, sdsPlot]);

  return (
    <>
      <SigPlotPanel
        href={rawHref}
        layerOptions={RAW_LAYER_OPTIONS}
        title="Raw file"
        fileName={fileName}
        onPlot={setRawPlot}
      />
      <SigPlotPanel
        href={sdsHref}
        layerOptions={SDS_LAYER_OPTIONS}
        title="SDS tiled"
        fileName={fileName}
        onPlot={setSdsPlot}
      />
    </>
  );
}
