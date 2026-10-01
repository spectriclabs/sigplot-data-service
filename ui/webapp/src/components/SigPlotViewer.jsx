import React, { useRef, useEffect, useMemo } from 'react';
import { Plot } from 'sigplot';

function removeLayer(plot, layer) {
  if (!plot || layer == null) return;
  if (Array.isArray(layer)) {
    layer.forEach((entry) => removeLayer(plot, entry));
    return;
  }
  if (plot.remove_layer) plot.remove_layer(layer);
}

function SigPlotPanel({ href, layerOptions, options, plotRef, title }) {
  const containerRef = useRef(null);
  const latestHrefRef = useRef(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const plot = new Plot(containerRef.current, {
      all: true,
      expand: true,
      autol: 100,
      autohide_panbars: true,
      ...options,
    });
    plotRef.current = plot;
    return () => {
      latestHrefRef.current = null;
      if (plot.deoverlay) plot.deoverlay();
      if (containerRef.current) containerRef.current.replaceChildren();
      plotRef.current = null;
    };
  }, []);

  useEffect(() => {
    const plot = plotRef.current;
    if (!plot) return;
    latestHrefRef.current = href;
    plot.deoverlay();
    if (href) {
      const layer = plot.overlay_href(
        href,
        () => {
          if (latestHrefRef.current !== href) removeLayer(plot, layer);
        },
        layerOptions && { ...layerOptions }
      );
    }
    return () => {
      if (latestHrefRef.current === href) latestHrefRef.current = null;
    };
  }, [href, layerOptions]);

  return (
    <div className="plot-container">
      <h4>{title}</h4>
      <div ref={containerRef} style={{ width: 550, height: 350 }} />
    </div>
  );
}

export default function SigPlotViewer({ rawHref, sdsHref }) {
  const rawPlotRef = useRef(null);
  const sdsPlotRef = useRef(null);
  const sdsLayerOptions = useMemo(() => ({ layerType: 'SDS' }), []);

  return (
    <>
      <SigPlotPanel
        href={rawHref}
        plotRef={rawPlotRef}
        title="Raw File"
      />
      <SigPlotPanel
        href={sdsHref}
        plotRef={sdsPlotRef}
        layerOptions={sdsLayerOptions}
        title="SDS Tiled View"
      />
    </>
  );
}
